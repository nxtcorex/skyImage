package api

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"skyimage/internal/admin"
	"skyimage/internal/captcha"
	"skyimage/internal/data"
	"skyimage/internal/files"
	mailservice "skyimage/internal/mail"
	"skyimage/internal/middleware"
	"skyimage/internal/notifications"
	"skyimage/internal/redeem"
	"skyimage/internal/users"
)

const defaultConsoleURL = "http://localhost:8080"

func (s *Server) registerAdminRoutes(r *gin.RouterGroup) {
	adminGroup := r.Group("/admin")
	adminGroup.Use(s.authMiddleware(), middleware.RequireAdmin(), middleware.RequireCSRF())
	adminGroup.GET("/metrics", s.handleAdminMetrics)
	adminGroup.GET("/trends", s.handleAdminTrends)
	adminGroup.GET("/settings", s.handleAdminSettings)
	adminGroup.PUT("/settings", s.handleAdminUpdateSettings)
	adminGroup.GET("/users", s.handleAdminUsers)
	adminGroup.GET("/users/:id", s.handleAdminGetUser)
	adminGroup.POST("/users", s.handleAdminCreateUser)
	adminGroup.DELETE("/users/:id", s.handleAdminDeleteUser)
	adminGroup.PATCH("/users/:id/status", s.handleAdminUpdateStatus)
	adminGroup.POST("/users/:id/admin", s.handleAdminToggleAdmin)
	adminGroup.PATCH("/users/:id/group", s.handleAdminAssignUserGroup)
	adminGroup.PATCH("/users/:id/capacity-bonus", s.handleAdminAdjustCapacityBonus)

	adminGroup.GET("/groups", s.handleAdminListGroups)
	adminGroup.POST("/groups", s.handleAdminCreateGroup)
	adminGroup.PUT("/groups/:id", s.handleAdminUpdateGroup)
	adminGroup.DELETE("/groups/:id", s.handleAdminDeleteGroup)

	adminGroup.GET("/redeem-codes", s.handleAdminListRedeemCodes)
	adminGroup.POST("/redeem-codes", s.handleAdminCreateRedeemCode)
	adminGroup.PUT("/redeem-codes/:id", s.handleAdminUpdateRedeemCode)
	adminGroup.DELETE("/redeem-codes/:id", s.handleAdminDeleteRedeemCode)
	adminGroup.GET("/redeem-codes/:id/usages", s.handleAdminListRedeemCodeUsages)

	adminGroup.GET("/strategies", s.handleAdminListStrategies)
	adminGroup.POST("/strategies", s.handleAdminCreateStrategy)
	adminGroup.PUT("/strategies/:id", s.handleAdminUpdateStrategy)
	adminGroup.DELETE("/strategies/:id", s.handleAdminDeleteStrategy)
	adminGroup.GET("/audits", s.handleAdminListAuditProfiles)
	adminGroup.POST("/audits", s.handleAdminCreateAuditProfile)
	adminGroup.PUT("/audits/:id", s.handleAdminUpdateAuditProfile)
	adminGroup.DELETE("/audits/:id", s.handleAdminDeleteAuditProfile)

	adminGroup.GET("/images", s.handleAdminImages)
	adminGroup.DELETE("/images/:id", s.handleAdminDeleteImage)
	adminGroup.PATCH("/images/:id/visibility", s.handleAdminUpdateImageVisibility)
	adminGroup.PATCH("/images/:id/audit-status", s.handleAdminUpdateImageAuditStatus)
	adminGroup.PATCH("/images/batch/visibility", s.handleAdminBatchUpdateImageVisibility)
	adminGroup.POST("/images/batch/delete", s.handleAdminBatchDeleteImages)

	adminGroup.GET("/system/site", s.handleAdminSiteSettings)
	adminGroup.PATCH("/system/site", s.handleAdminUpdateSiteSettings)
	adminGroup.GET("/system/site/legal/:type", s.handleAdminSiteLegal)
	adminGroup.PUT("/system/site/legal/:type", s.handleAdminUpdateSiteLegal)
	adminGroup.GET("/system/general", s.handleAdminGeneralSettings)
	adminGroup.PATCH("/system/general", s.handleAdminUpdateGeneralSettings)
	adminGroup.GET("/system/tickets", s.handleAdminTicketSettings)
	adminGroup.PATCH("/system/tickets", s.handleAdminUpdateTicketSettings)
	adminGroup.GET("/system/email", s.handleAdminEmailSettings)
	adminGroup.PATCH("/system/email", s.handleAdminUpdateEmailSettings)
	adminGroup.POST("/system/email/test", s.handleAdminTestSMTP)
	adminGroup.GET("/system/captcha", s.handleAdminCaptchaSettings)
	adminGroup.PATCH("/system/captcha", s.handleAdminUpdateCaptchaSettings)
	adminGroup.POST("/system/captcha/test-turnstile", s.handleAdminTestTurnstile)
	adminGroup.POST("/system/captcha/test", s.handleAdminTestCaptcha)
	adminGroup.GET("/system/oauth", s.handleAdminOAuthSettings)
	adminGroup.PATCH("/system/oauth", s.handleAdminUpdateOAuthSettings)
	adminGroup.GET("/system/database", s.handleAdminDatabaseConfig)
	adminGroup.POST("/system/database/test", s.handleAdminDatabaseTest)
	adminGroup.POST("/system/database/migrate", s.handleAdminDatabaseMigrate)
	s.registerAdminShopRoutes(adminGroup)
	s.registerAdminTicketRoutes(adminGroup)
}

func requireSuperAdmin(c *gin.Context) bool {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return false
	}
	if !user.IsSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "super admin required"})
		return false
	}
	return true
}

func redactSettings(settings map[string]string) map[string]string {
	if len(settings) == 0 {
		return settings
	}
	redacted := make(map[string]string, len(settings))
	for key, value := range settings {
		redacted[key] = value
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "mail.smtp.password",
			"captcha.cloudflare.secret_key", "captcha.cloudflare.last_verified_signature",
			"captcha.geetest.captcha_key", "captcha.geetest.last_verified_signature",
			"captcha.cap.secret_key", "captcha.cap.last_verified_signature":
			redacted[key] = "***"
		}
	}
	return redacted
}

// redactSecret returns "***" for non-empty secrets, empty string otherwise.
func redactSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "***"
}

func (s *Server) handleAdminMetrics(c *gin.Context) {
	metrics, err := s.admin.Dashboard(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

func (s *Server) handleAdminTrends(c *gin.Context) {
	days := 90
	if daysParam := c.Query("days"); daysParam != "" {
		if parsedDays, err := strconv.Atoi(daysParam); err == nil && parsedDays > 0 {
			days = parsedDays
		}
	}

	trends, err := s.admin.GetTrends(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": trends})
}

func (s *Server) handleAdminUsers(c *gin.Context) {
	users, err := s.users.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func parseRouteUserID(c *gin.Context) (uint, error) {
	return data.ParseUserID(c.Param("id"))
}

func (s *Server) handleAdminGetUser(c *gin.Context) {
	id, err := parseRouteUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	user, err := s.users.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (s *Server) handleAdminUpdateStatus(c *gin.Context) {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := parseRouteUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		Status uint8 `json:"status"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 演示站模式：禁止修改演示账户状态
	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	adminEmail := strings.ToLower(strings.TrimSpace(s.cfg.AdminEmail))
	demoUserEmail := strings.ToLower(strings.TrimSpace(s.cfg.DemoUserEmail))
	s.mu.RUnlock()

	if demoMode {
		target, findErr := s.users.FindByID(c.Request.Context(), id)
		if findErr == nil {
			targetEmail := strings.ToLower(strings.TrimSpace(target.Email))
			if targetEmail == adminEmail || targetEmail == demoUserEmail {
				c.JSON(http.StatusForbidden, gin.H{"error": "演示站禁止修改此账户状态"})
				return
			}
		}
	}

	if err := s.users.UpdateStatus(c.Request.Context(), actor, id, payload.Status); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "updated"})
}

func (s *Server) handleAdminToggleAdmin(c *gin.Context) {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := parseRouteUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		Admin bool `json:"admin"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 演示站模式：禁止修改演示账户角色
	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	adminEmail := strings.ToLower(strings.TrimSpace(s.cfg.AdminEmail))
	demoUserEmail := strings.ToLower(strings.TrimSpace(s.cfg.DemoUserEmail))
	s.mu.RUnlock()

	if demoMode {
		target, findErr := s.users.FindByID(c.Request.Context(), id)
		if findErr == nil {
			targetEmail := strings.ToLower(strings.TrimSpace(target.Email))
			if targetEmail == adminEmail || targetEmail == demoUserEmail {
				c.JSON(http.StatusForbidden, gin.H{"error": "演示站禁止修改此账户角色"})
				return
			}
		}
	}

	if err := s.users.ToggleAdmin(c.Request.Context(), actor, id, payload.Admin); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "updated"})
}

func (s *Server) handleAdminAssignUserGroup(c *gin.Context) {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := parseRouteUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		GroupID *uint `json:"groupId"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 演示站模式：禁止修改演示账户角色组
	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	adminEmail := strings.ToLower(strings.TrimSpace(s.cfg.AdminEmail))
	demoUserEmail := strings.ToLower(strings.TrimSpace(s.cfg.DemoUserEmail))
	s.mu.RUnlock()

	if demoMode {
		target, findErr := s.users.FindByID(c.Request.Context(), id)
		if findErr == nil {
			targetEmail := strings.ToLower(strings.TrimSpace(target.Email))
			if targetEmail == adminEmail || targetEmail == demoUserEmail {
				c.JSON(http.StatusForbidden, gin.H{"error": "演示站禁止修改此账户角色组"})
				return
			}
		}
	}

	user, err := s.users.AssignGroup(c.Request.Context(), actor, id, payload.GroupID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (s *Server) handleAdminAdjustCapacityBonus(c *gin.Context) {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := parseRouteUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		// deltaBytes: 相对当前 bonus 增减（字节，可负）
		DeltaBytes *float64 `json:"deltaBytes"`
		// bonusBytes: 直接设置 bonus（字节）；与 delta 二选一，优先 delta
		BonusBytes *float64 `json:"bonusBytes"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if payload.DeltaBytes == nil && payload.BonusBytes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deltaBytes or bonusBytes required"})
		return
	}

	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	s.mu.RUnlock()
	if demoMode {
		c.JSON(http.StatusForbidden, gin.H{"error": "演示站禁止修改用户容量"})
		return
	}

	var user data.User
	if payload.DeltaBytes != nil {
		user, err = s.users.AdjustCapacityBonus(c.Request.Context(), actor, id, *payload.DeltaBytes)
	} else {
		user, err = s.users.SetCapacityBonus(c.Request.Context(), actor, id, *payload.BonusBytes)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (s *Server) handleAdminCreateUser(c *gin.Context) {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	var payload users.CreateUserInput
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := s.users.CreateUser(c.Request.Context(), actor, payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (s *Server) handleAdminDeleteUser(c *gin.Context) {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := parseRouteUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// 演示站模式：禁止删除演示账户
	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	adminEmail := strings.ToLower(strings.TrimSpace(s.cfg.AdminEmail))
	demoUserEmail := strings.ToLower(strings.TrimSpace(s.cfg.DemoUserEmail))
	s.mu.RUnlock()

	if demoMode {
		target, findErr := s.users.FindByID(c.Request.Context(), id)
		if findErr == nil {
			targetEmail := strings.ToLower(strings.TrimSpace(target.Email))
			if targetEmail == adminEmail || targetEmail == demoUserEmail {
				c.JSON(http.StatusForbidden, gin.H{"error": "演示站禁止删除此账户"})
				return
			}
		}
	}

	if err := s.users.DeleteUser(c.Request.Context(), actor, id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "deleted"})
}

func (s *Server) handleAdminListGroups(c *gin.Context) {
	groups, err := s.admin.ListGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": groups})
}

func (s *Server) handleAdminCreateGroup(c *gin.Context) {
	var payload admin.GroupPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	group, err := s.admin.CreateGroup(c.Request.Context(), payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": group})
}

func (s *Server) handleAdminUpdateGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// 演示站模式：禁止修改角色组配置
	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	s.mu.RUnlock()

	if demoMode {
		c.JSON(http.StatusForbidden, gin.H{"error": "演示站禁止修改角色组配置"})
		return
	}

	var payload admin.GroupPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	group, err := s.admin.UpdateGroup(c.Request.Context(), uint(id), payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": group})
}

func (s *Server) handleAdminDeleteGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// 演示站模式：禁止删除角色组
	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	s.mu.RUnlock()

	if demoMode {
		c.JSON(http.StatusForbidden, gin.H{"error": "演示站禁止删除角色组"})
		return
	}

	if err := s.admin.DeleteGroup(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "deleted"})
}

func (s *Server) handleAdminListStrategies(c *gin.Context) {
	items, err := s.admin.ListStrategies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (s *Server) handleAdminCreateStrategy(c *gin.Context) {
	// 演示站模式检查：禁止创建存储策略
	if s.cfg.DemoMode {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "演示站禁止创建存储策略",
			"message": "演示站环境不允许添加新的存储策略配置，所有数据仅保存在容器内临时存储中",
		})
		return
	}

	var payload admin.StrategyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := s.admin.CreateStrategy(c.Request.Context(), payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (s *Server) handleAdminUpdateStrategy(c *gin.Context) {
	// 演示站模式检查：禁止更新存储策略
	if s.cfg.DemoMode {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "演示站禁止更新存储策略",
			"message": "演示站环境不允许修改存储策略配置，所有数据仅保存在容器内临时存储中",
		})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	existing, err := s.admin.FindStrategyByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	var payload admin.StrategyPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.files.FreezePublicURLsForStrategy(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	item, err := s.admin.UpdateStrategy(c.Request.Context(), uint(id), payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (s *Server) handleAdminDeleteStrategy(c *gin.Context) {
	// 演示站模式检查：禁止删除存储策略
	if s.cfg.DemoMode {
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "演示站禁止删除存储策略",
			"message": "演示站环境不允许删除存储策略配置",
		})
		return
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := s.admin.DeleteStrategy(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "deleted"})
}

func (s *Server) handleAdminListAuditProfiles(c *gin.Context) {
	items, err := s.admin.ListAuditProfiles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (s *Server) handleAdminCreateAuditProfile(c *gin.Context) {
	var payload admin.AuditProfilePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := s.admin.CreateAuditProfile(c.Request.Context(), payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (s *Server) handleAdminUpdateAuditProfile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload admin.AuditProfilePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := s.admin.UpdateAuditProfile(c.Request.Context(), uint(id), payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (s *Server) handleAdminDeleteAuditProfile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	err = s.admin.DeleteAuditProfile(c.Request.Context(), uint(id))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, admin.ErrAuditProfileInUse) {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "deleted"})
}

func (s *Server) handleAdminImages(c *gin.Context) {
	limit, offset := parsePagination(c, 50, 100)
	auditStatus := strings.TrimSpace(c.Query("auditStatus"))
	filesList, err := s.admin.ListAllFiles(c.Request.Context(), limit, offset, auditStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	viewer, _ := middleware.CurrentUser(c)
	dtos := make([]files.FileDTO, 0, len(filesList))
	for _, file := range filesList {
		dto, err := s.files.ToDTOForViewer(c.Request.Context(), file, &viewer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		dtos = append(dtos, dto)
	}
	c.JSON(http.StatusOK, gin.H{"data": dtos})
}

func (s *Server) handleAdminDeleteImage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil && err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.files.DeleteByAdmin(c.Request.Context(), uint(id), payload.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "deleted"})
}

func (s *Server) handleAdminUpdateImageVisibility(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		Visibility string `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	file, err := s.files.UpdateVisibilityByAdmin(c.Request.Context(), uint(id), payload.Visibility)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	viewer, _ := middleware.CurrentUser(c)
	dto, err := s.files.ToDTOForViewer(c.Request.Context(), file, &viewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dto})
}

func (s *Server) handleAdminUpdateImageAuditStatus(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	file, err := s.files.UpdateAuditStatusByAdmin(c.Request.Context(), uint(id), payload.Status)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	viewer, _ := middleware.CurrentUser(c)
	dto, err := s.files.ToDTOForViewer(c.Request.Context(), file, &viewer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dto})
}

func (s *Server) handleAdminBatchUpdateImageVisibility(c *gin.Context) {
	var payload struct {
		IDs        []uint `json:"ids"`
		Visibility string `json:"visibility"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	updated, err := s.files.UpdateVisibilityByAdminBatch(c.Request.Context(), payload.IDs, payload.Visibility)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"updated": updated}})
}

func (s *Server) handleAdminBatchDeleteImages(c *gin.Context) {
	var payload struct {
		IDs    []uint `json:"ids"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	deleted, err := s.files.DeleteByAdminBatch(c.Request.Context(), payload.IDs, payload.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"deleted": deleted}})
}

func (s *Server) handleAdminSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": redactSettings(settings)})
}

func (s *Server) handleAdminUpdateSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload map[string]string
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.admin.UpdateSettings(c.Request.Context(), payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "ok"})
}

func normalizeImageLoadRows(raw string) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		value = 4
	}
	return normalizeImageLoadRowsValue(value)
}

func normalizeImageLoadRowsValue(value int) int {
	if value < 1 {
		return 1
	}
	if value > 20 {
		return 20
	}
	return value
}

func normalizeUserNotificationLimit(value int) int {
	return notifications.NormalizeRetentionLimitValue(value)
}

type testTurnstilePayload struct {
	SiteKey   string `json:"siteKey" binding:"required"`
	SecretKey string `json:"secretKey" binding:"required"`
	Token     string `json:"token" binding:"required"`
}

func (s *Server) handleAdminTestTurnstile(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload testTurnstilePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先填写完整的 Turnstile 配置信息并通过验证"})
		return
	}
	// Use the Cloudflare service from the unified captcha service
	ok, err := s.captcha.Verify(c.Request.Context(), captcha.ProviderCloudflare, payload.Token, getClientIP(c, s.isCDNEnabled(c.Request.Context())), nil)
	if err != nil || !ok {
		message := "Turnstile 验证失败"
		if err != nil {
			message = err.Error()
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"success": false,
			"message": message,
		}})
		return
	}
	// Resolve "***" to actual secret key from database for signature calculation
	secretKey := strings.TrimSpace(payload.SecretKey)
	if secretKey == "***" {
		if settings, err := s.admin.GetSettings(c.Request.Context()); err == nil {
			secretKey = settings["turnstile.secret_key"]
		}
	}
	signature := captcha.GenerateSignature(payload.SiteKey, secretKey)
	now := time.Now().UTC().Format(time.RFC3339)
	if err := s.admin.UpdateSettings(c.Request.Context(), map[string]string{
		"turnstile.last_verified_signature": signature,
		"turnstile.last_verified_at":        now,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"success":    true,
		"verifiedAt": now,
	}})
}

type testSMTPPayload struct {
	TestEmail       string `json:"testEmail" binding:"required,email"`
	SiteTitle       string `json:"siteTitle"`
	SMTPHost        string `json:"smtpHost" binding:"required"`
	SMTPPort        string `json:"smtpPort" binding:"required"`
	SMTPUsername    string `json:"smtpUsername" binding:"required"`
	SMTPPassword    string `json:"smtpPassword"`
	SMTPFrom        string `json:"smtpFrom"`
	SMTPSecure      bool   `json:"smtpSecure"`
	MailTestSubject string `json:"mailTestSubject"`
	MailTestBody    string `json:"mailTestBody"`
}

func (s *Server) handleAdminTestSMTP(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload testSMTPPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写完整的邮件配置信息"})
		return
	}

	siteTitle := strings.TrimSpace(payload.SiteTitle)
	if siteTitle == "" {
		settings, err := s.admin.GetSettings(c.Request.Context())
		if err == nil {
			siteTitle = settings["site.title"]
		}
	}
	template := mailservice.RenderTemplateContent(
		mailservice.MergeTemplateContent(mailservice.TemplateTestSMTP, payload.MailTestSubject, payload.MailTestBody),
		mailservice.TemplateVariables{
			SiteName:  siteTitle,
			TestEmail: payload.TestEmail,
		},
	)

	fromRaw := strings.TrimSpace(payload.SMTPFrom)
	if fromRaw == "" {
		fromRaw = payload.SMTPUsername
	}

	cfg := &mailservice.SMTPConfig{
		Host:     strings.TrimSpace(payload.SMTPHost),
		Port:     strings.TrimSpace(payload.SMTPPort),
		Username: payload.SMTPUsername,
		Password: payload.SMTPPassword,
		From:     fromRaw,
		Secure:   payload.SMTPSecure,
	}

	if err := s.mail.SendMailWithConfig(cfg, payload.TestEmail, template.Subject, template.Body); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"success": false,
				"message": "发送邮件失败: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"success": true,
			"message": "测试邮件发送成功",
		},
	})
}

type testCaptchaPayload struct {
	Provider    string            `json:"provider" binding:"required"`
	SiteKey     string            `json:"siteKey"`
	SecretKey   string            `json:"secretKey"`
	CaptchaID   string            `json:"captchaId"`
	CaptchaKey  string            `json:"captchaKey"`
	InstanceURL string            `json:"instanceUrl"`
	Token       string            `json:"token" binding:"required"`
	ExtraData   map[string]string `json:"extraData"`
}

func (s *Server) handleAdminTestCaptcha(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}

	var payload testCaptchaPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请填写完整的验证码配置信息"})
		return
	}

	provider := captcha.Provider(payload.Provider)
	config := map[string]string{}

	// Get current settings for resolving redacted secrets
	settings, _ := s.admin.GetSettings(c.Request.Context())

	if provider == captcha.ProviderCloudflare {
		secretKey := strings.TrimSpace(payload.SecretKey)
		if secretKey == "***" && settings != nil {
			secretKey = settings["captcha.cloudflare.secret_key"]
		}
		if payload.SiteKey == "" || secretKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cloudflare Turnstile 需要 Site Key 和 Secret Key"})
			return
		}
		config["secret_key"] = secretKey
	} else if provider == captcha.ProviderGeetest {
		captchaKey := strings.TrimSpace(payload.CaptchaKey)
		if captchaKey == "***" && settings != nil {
			captchaKey = settings["captcha.geetest.captcha_key"]
		}
		if payload.CaptchaID == "" || captchaKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "极验需要 Captcha ID 和 Captcha Key"})
			return
		}
		config["captcha_id"] = payload.CaptchaID
		config["captcha_key"] = captchaKey
	} else if provider == captcha.ProviderCap {
		secretKey := strings.TrimSpace(payload.SecretKey)
		if secretKey == "***" && settings != nil {
			secretKey = settings["captcha.cap.secret_key"]
		}
		instanceURL := strings.TrimSpace(payload.InstanceURL)
		if instanceURL == "" && settings != nil {
			instanceURL = settings["captcha.cap.instance_url"]
		}
		siteKey := strings.TrimSpace(payload.SiteKey)
		if siteKey == "" && settings != nil {
			siteKey = settings["captcha.cap.site_key"]
		}
		if instanceURL == "" || siteKey == "" || secretKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cap 需要 Instance URL、Site Key 和 Secret Key"})
			return
		}
		normalized, err := captcha.NormalizeCapInstanceURL(instanceURL)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		config["instance_url"] = normalized
		config["site_key"] = siteKey
		config["secret_key"] = secretKey
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的验证码提供商"})
		return
	}

	ok, err := s.captcha.TestConfig(c.Request.Context(), provider, config, payload.Token, payload.ExtraData)
	if err != nil || !ok {
		message := "验证失败"
		if err != nil {
			message = err.Error()
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"success": false,
			"message": message,
		}})
		return
	}

	// 保存验证通过的状态（使用解析后的真实密钥计算签名）
	now := time.Now().UTC().Format(time.RFC3339)
	settingsUpdate := map[string]string{}

	if provider == captcha.ProviderCloudflare {
		signature := captcha.GenerateSignature(payload.SiteKey, config["secret_key"])
		settingsUpdate["captcha.cloudflare.last_verified_signature"] = signature
		settingsUpdate["captcha.cloudflare.last_verified_at"] = now
	} else if provider == captcha.ProviderGeetest {
		signature := captcha.GenerateGeetestSignature(payload.CaptchaID, config["captcha_key"])
		settingsUpdate["captcha.geetest.last_verified_signature"] = signature
		settingsUpdate["captcha.geetest.last_verified_at"] = now
	} else if provider == captcha.ProviderCap {
		signature := captcha.GenerateCapSignature(config["instance_url"], config["site_key"], config["secret_key"])
		settingsUpdate["captcha.cap.last_verified_signature"] = signature
		settingsUpdate["captcha.cap.last_verified_at"] = now
	}

	if err := s.admin.UpdateSettings(c.Request.Context(), settingsUpdate); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"success":    true,
		"message":    "验证成功",
		"verifiedAt": now,
	}})
}

func (s *Server) handleAdminListRedeemCodes(c *gin.Context) {
	items, err := s.redeem.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (s *Server) handleAdminCreateRedeemCode(c *gin.Context) {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}

	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	s.mu.RUnlock()
	if demoMode {
		c.JSON(http.StatusForbidden, gin.H{"error": "演示站禁止创建兑换码"})
		return
	}

	var payload redeem.CreateInput
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := s.redeem.Create(c.Request.Context(), actor, payload)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, redeem.ErrAdminRequired) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (s *Server) handleAdminUpdateRedeemCode(c *gin.Context) {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	s.mu.RUnlock()
	if demoMode {
		c.JSON(http.StatusForbidden, gin.H{"error": "演示站禁止修改兑换码"})
		return
	}

	var payload redeem.UpdateInput
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := s.redeem.Update(c.Request.Context(), actor, uint(id), payload)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, redeem.ErrCodeNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, redeem.ErrAdminRequired) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (s *Server) handleAdminDeleteRedeemCode(c *gin.Context) {
	actor, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	s.mu.RUnlock()
	if demoMode {
		c.JSON(http.StatusForbidden, gin.H{"error": "演示站禁止删除兑换码"})
		return
	}

	if err := s.redeem.Delete(c.Request.Context(), actor, uint(id)); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, redeem.ErrCodeNotFound) {
			status = http.StatusNotFound
		} else if errors.Is(err, redeem.ErrAdminRequired) {
			status = http.StatusForbidden
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "deleted"})
}

func (s *Server) handleAdminListRedeemCodeUsages(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if _, err := s.redeem.Get(c.Request.Context(), uint(id)); err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, redeem.ErrCodeNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	items, err := s.redeem.ListUsages(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}
