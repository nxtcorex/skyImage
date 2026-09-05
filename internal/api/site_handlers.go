package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"skyimage/internal/captcha"
	"skyimage/internal/data"
	"skyimage/internal/files"
	"skyimage/internal/middleware"
	"skyimage/internal/users"
)

func (s *Server) registerSiteRoutes(r *gin.RouterGroup) {
	r.GET("/site/config", s.handleSiteConfig)
	r.GET("/site/turnstile/:scenario", s.handleTurnstileConfig)
	r.GET("/gallery/public", middleware.OptionalAuth(s.users, s.session), s.handleGalleryPublic)
	r.GET("/users/:id/public", middleware.OptionalAuth(s.users, s.session), s.handlePublicUserProfile)
	s.engine.GET("/favicon.ico", s.handleFavicon)
}

func (s *Server) handleSiteConfig(c *gin.Context) {
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	status, err := s.installer.Status(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	enableGallery := settings["features.gallery"] != "false"
	enableHome := settings["features.home"] != "false"
	enableAPI := settings["features.api"] != "false"
	disabledNotice := settings["account.disabled_notice"]
	if strings.TrimSpace(disabledNotice) == "" {
		disabledNotice = defaultAccountDisabledNotice
	}

	aboutText := settings["site.about"]
	if strings.TrimSpace(aboutText) == "" {
		aboutText = status.About
	}
	homePageMode := strings.TrimSpace(settings["site.home_page_mode"])
	if homePageMode != "custom_html" {
		homePageMode = "default"
	}
	homeCustomHTML := ""
	if homePageMode == "custom_html" {
		homeCustomHTML = settings["site.home_custom_html"]
	}

	response := gin.H{
		"title":                          settings["site.title"],
		"description":                    settings["site.description"],
		"slogan":                         settings["site.slogan"],
		"logo":                           settings["site.logo"],
		"about":                          aboutText,
		"aboutTitle":                     settings["site.about_title"],
		"notFoundMode":                   settings["site.notfound_mode"],
		"notFoundHeading":                settings["site.notfound_heading"],
		"notFoundText":                   settings["site.notfound_text"],
		"notFoundHtml":                   settings["site.notfound_html"],
		"termsOfService":                 settings["site.terms_of_service"],
		"privacyPolicy":                  settings["site.privacy_policy"],
		"homePageMode":                   homePageMode,
		"homeCustomHtml":                 homeCustomHTML,
		"enableGallery":                  enableGallery,
		"enableHome":                     enableHome,
		"enableApi":                      enableAPI,
		"imageLoadRows":                  normalizeImageLoadRows(settings["images.load_rows"]),
		"forgotPasswordEnabled":          settings["mail.forgot_password.enabled"] == "true",
		"forgotPasswordTurnstileRequest": settings["mail.forgot_password.turnstile_request"] == "true",
		"forgotPasswordTurnstileReset":   settings["mail.forgot_password.turnstile_reset"] == "true",
		"version":                        status.Version,
		"accountDisabledNotice":          disabledNotice,
		"hiddenSidebarItems":             splitConfigList(settings[ConfigSidebarHidden]),
	}
	c.JSON(http.StatusOK, gin.H{"data": response})
}

func (s *Server) handleGalleryPublic(c *gin.Context) {
	limit, offset := parsePagination(c, 40, 100)
	items, err := s.files.ListPublic(c.Request.Context(), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var viewer *data.User
	if user, ok := middleware.CurrentUser(c); ok {
		viewer = &user
	}
	dtos := make([]files.FileDTO, 0, len(items))
	for _, file := range items {
		dto, err := s.files.ToDTOForViewer(c.Request.Context(), file, viewer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		dto.Audit = nil
		// Gallery is public; do not leak owner emails.
		dto.OwnerEmail = ""
		// Only expose profile link target when the owner enabled public profile.
		if !dto.OwnerPublicProfile {
			dto.OwnerID = 0
		}
		dtos = append(dtos, dto)
	}
	c.JSON(http.StatusOK, gin.H{"data": dtos})
}

func (s *Server) handlePublicUserProfile(c *gin.Context) {
	userID, err := data.ParseUserID(c.Param("id"))
	if err != nil || userID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	user, err := s.users.FindByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if user.Status == 0 || !users.PublicProfileEnabled(user) {
		// Closed profile: not visible to anyone (including owner).
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	avatarURL := ""
	var binding data.UserOAuthBinding
	if err := s.db.WithContext(c.Request.Context()).
		Where("user_id = ? AND avatar_url <> ''", user.ID).
		Order("updated_at DESC").
		First(&binding).Error; err == nil {
		avatarURL = binding.AvatarURL
	}

	limit, offset := parsePagination(c, 40, 100)
	items, err := s.files.ListPublicByUser(c.Request.Context(), user.ID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	images := make([]gin.H, 0, len(items))
	for _, file := range items {
		viewURL, err := s.files.PublicURL(c.Request.Context(), file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// Thumbnails are login-only in this app; public profile exposes the image URL for both.
		images = append(images, gin.H{
			"viewUrl":      viewURL,
			"thumbnailUrl": viewURL,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"name":      user.Name,
			"avatarUrl": avatarURL,
			"images":    images,
		},
	})
}

func (s *Server) handleTurnstileConfig(c *gin.Context) {
	scenario := c.Param("scenario")
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Use new captcha.* keys (migration from turnstile.* is handled at startup)
	var configKey string
	switch scenario {
	case "login":
		configKey = "captcha.login"
	case "register":
		configKey = "captcha.register"
	case "register_verify":
		configKey = "captcha.register_verify"
	case "forgot_password_request":
		configKey = "captcha.forgot_password_request"
	case "redeem":
		configKey = "captcha.redeem"
	case "ticket":
		configKey = "captcha.ticket"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scenario"})
		return
	}

	enabled := settings["captcha.enabled"] == "true" && settings[configKey] == "true"

	response := gin.H{
		"enabled": enabled,
	}

	if enabled {
		provider := settings["captcha.provider"]
		var siteKey string
		if provider == "cloudflare" {
			siteKey = settings["captcha.cloudflare.site_key"]
		} else if provider == "geetest" {
			siteKey = settings["captcha.geetest.captcha_id"]
		} else if provider == "cap" {
			siteKey = settings["captcha.cap.site_key"]
			if endpoint, err := captcha.BuildCapAPIEndpoint(settings["captcha.cap.instance_url"], siteKey); err == nil {
				response["apiEndpoint"] = endpoint
			}
		}
		if siteKey != "" {
			response["siteKey"] = siteKey
			response["provider"] = provider
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

func (s *Server) handleFavicon(c *gin.Context) {
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	logoURL := strings.TrimSpace(settings["site.logo"])
	if logoURL == "" {
		c.Status(http.StatusNotFound)
		return
	}

	// 如果是外部链接，重定向到该链接
	if strings.HasPrefix(logoURL, "http://") || strings.HasPrefix(logoURL, "https://") {
		c.Redirect(http.StatusFound, logoURL)
		return
	}

	// 如果是相对路径，重定向到实际的文件URL
	// 这样可以利用现有的文件服务逻辑
	if !strings.HasPrefix(logoURL, "/") {
		logoURL = "/" + logoURL
	}
	c.Redirect(http.StatusFound, logoURL)
}
