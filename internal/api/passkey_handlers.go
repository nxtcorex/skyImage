package api

import (
	"errors"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"skyimage/internal/middleware"
	"skyimage/internal/passkey"
	"skyimage/internal/users"
)

func (s *Server) registerPasskeyRoutes(r *gin.RouterGroup) {
	pk := r.Group("/auth/passkeys")

	// Public ceremony endpoints (usernameless login).
	pk.POST("/login/begin", s.handlePasskeyLoginBegin)
	pk.POST("/login/complete", s.handlePasskeyLoginComplete)

	// Authenticated ceremony + management endpoints (middleware applies to the
	// routes registered after this point).
	pk.Use(s.authMiddleware())
	pk.POST("/register/begin", middleware.RequireCSRF(), s.handlePasskeyRegisterBegin)
	pk.POST("/register/complete", middleware.RequireCSRF(), s.handlePasskeyRegisterComplete)
	pk.GET("", s.handlePasskeyList)
	pk.PATCH("/:id", middleware.RequireCSRF(), s.handlePasskeyRename)
	pk.DELETE("/:id", middleware.RequireCSRF(), s.handlePasskeyDelete)
}

// passkeySiteName resolves the display name for the Relying Party.
func (s *Server) passkeySiteName(c *gin.Context) string {
	if settings, err := s.admin.GetSettings(c.Request.Context()); err == nil {
		if title := strings.TrimSpace(settings["site.title"]); title != "" {
			return title
		}
	}
	if strings.TrimSpace(s.cfg.SiteName) != "" {
		return s.cfg.SiteName
	}
	return ""
}

// requestOrigin reconstructs the caller's origin (scheme + host) without
// trusting the Host header beyond origin matching (RPID stays config-driven).
func requestOrigin(c *gin.Context) string {
	host := c.Request.Host
	if strings.TrimSpace(host) == "" {
		return ""
	}
	scheme := "http"
	if isSecureRequest(c) {
		scheme = "https"
	}
	return scheme + "://" + host
}

func (s *Server) passkeyEnabled(c *gin.Context) bool {
	return s.passkeys != nil && s.passkeys.Enabled(c.Request.Context())
}

func (s *Server) handlePasskeyRegisterBegin(c *gin.Context) {
	if !s.passkeyEnabled(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "passkeys disabled"})
		return
	}
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	options, err := s.passkeys.BeginRegistration(c.Request.Context(), user.ID, requestOrigin(c), s.passkeySiteName(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": options})
}

func (s *Server) handlePasskeyRegisterComplete(c *gin.Context) {
	if !s.passkeyEnabled(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "passkeys disabled"})
		return
	}
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || len(raw) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	entry, err := s.passkeys.FinishRegistration(c.Request.Context(), user.ID, requestOrigin(c), s.passkeySiteName(c), raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dto, err := s.passkeys.ToDTO(*entry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to serialize passkey"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dto})
}

func (s *Server) handlePasskeyLoginBegin(c *gin.Context) {
	if !s.passkeyEnabled(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "passkeys disabled"})
		return
	}
	clientIP := getClientIP(c, s.isCDNEnabled(c.Request.Context()))
	if ok, retry := s.authLimiter.Allow("passkey:login:ip:"+clientIP, 30, time.Minute); !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试", "retryAfterSeconds": int(retry.Seconds()) + 1})
		return
	}
	assertion, err := s.passkeys.BeginLogin(c.Request.Context(), requestOrigin(c), s.passkeySiteName(c))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": assertion})
}

func (s *Server) handlePasskeyLoginComplete(c *gin.Context) {
	if !s.passkeyEnabled(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "passkeys disabled"})
		return
	}
	clientIP := getClientIP(c, s.isCDNEnabled(c.Request.Context()))
	if ok, retry := s.authLimiter.Allow("passkey:login:ip:"+clientIP, 30, time.Minute); !ok {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "请求过于频繁，请稍后再试", "retryAfterSeconds": int(retry.Seconds()) + 1})
		return
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || len(raw) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	user, _, err := s.passkeys.FinishLogin(c.Request.Context(), requestOrigin(c), s.passkeySiteName(c), raw)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	if user.Status == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "account disabled"})
		return
	}
	sessionID, err := s.session.Create(user.ID)
	if err != nil {
		log.Printf("[passkey] 创建会话失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}
	s.writeSessionCookie(c, sessionID)
	s.writeCSRFCookie(c)

	go func() {
		if s.mail == nil {
			return
		}
		userNotifyEnabled := users.LoginNotificationEnabled(user)
		if err := s.mail.SendLoginNotification(c.Request.Context(), user.Email, user.Name, clientIP, userNotifyEnabled); err != nil {
			log.Printf("[邮件] Passkey 登录提醒失败: %v", err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{"data": gin.H{"user": user}})
}

func (s *Server) handlePasskeyList(c *gin.Context) {
	if !s.passkeyEnabled(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "passkeys disabled"})
		return
	}
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	items, err := s.passkeys.List(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	dtos := make([]passkey.PasskeyDTO, 0, len(items))
	for _, item := range items {
		dto, err := s.passkeys.ToDTO(item)
		if err != nil {
			log.Printf("[passkey] failed to decode passkey %d: %v", item.ID, err)
			continue
		}
		dtos = append(dtos, dto)
	}
	c.JSON(http.StatusOK, gin.H{"data": dtos})
}

func (s *Server) handlePasskeyDelete(c *gin.Context) {
	if !s.passkeyEnabled(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "passkeys disabled"})
		return
	}
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey id"})
		return
	}
	if err := s.passkeys.Delete(c.Request.Context(), user.ID, uint(id)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "passkey not found"})
			return
		}
		log.Printf("[passkey] delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete passkey"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "deleted"}})
}

func (s *Server) handlePasskeyRename(c *gin.Context) {
	if !s.passkeyEnabled(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "passkeys disabled"})
		return
	}
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 || id > math.MaxUint {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey id"})
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" || len([]rune(name)) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid passkey name"})
		return
	}
	if err := s.passkeys.Rename(c.Request.Context(), user.ID, uint(id), name); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "passkey not found"})
			return
		}
		log.Printf("[passkey] rename failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rename passkey"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "renamed"}})
}
