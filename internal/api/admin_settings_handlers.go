package api

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"skyimage/internal/captcha"
	"skyimage/internal/notifications"
	"skyimage/internal/tickets"
)

func parseUintSetting(raw string) uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return uint(value)
}

// ---------------------------------------------------------------------------
// Site Settings (GET /admin/system/site, PATCH /admin/system/site)
// Legal texts (terms/privacy) live in their own endpoints: /admin/system/site/legal/:type
// ---------------------------------------------------------------------------

type siteSettingsPayload struct {
	SiteTitle             string `json:"siteTitle"`
	ConsoleURL            string `json:"consoleUrl"`
	SiteDescription       string `json:"siteDescription"`
	SiteSlogan            string `json:"siteSlogan"`
	SiteLogo              string `json:"siteLogo"`
	About                 string `json:"about"`
	AboutTitle            string `json:"aboutTitle"`
	NotFoundMode          string `json:"notFoundMode"`
	NotFoundHeading       string `json:"notFoundHeading"`
	NotFoundText          string `json:"notFoundText"`
	NotFoundHtml          string `json:"notFoundHtml"`
	HomePageMode          string `json:"homePageMode"`
	HomeCustomHTML        string `json:"homeCustomHtml"`
	AccountDisabledNotice string `json:"accountDisabledNotice"`
}

func (s *Server) handleAdminSiteSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	consoleURL := strings.TrimSpace(settings["site.console_url"])
	if consoleURL == "" {
		consoleURL = defaultConsoleURL
	}
	homePageMode := strings.TrimSpace(settings["site.home_page_mode"])
	if homePageMode != "custom_html" {
		homePageMode = "default"
	}
	homeCustomHTML := ""
	if homePageMode == "custom_html" {
		homeCustomHTML = settings["site.home_custom_html"]
	}
	disabledNotice := settings["account.disabled_notice"]
	if strings.TrimSpace(disabledNotice) == "" {
		disabledNotice = defaultAccountDisabledNotice
	}

	payload := siteSettingsPayload{
		SiteTitle:             settings["site.title"],
		ConsoleURL:            consoleURL,
		SiteDescription:       settings["site.description"],
		SiteSlogan:            settings["site.slogan"],
		SiteLogo:              settings["site.logo"],
		About:                 settings["site.about"],
		AboutTitle:            settings["site.about_title"],
		NotFoundMode:          settings["site.notfound_mode"],
		NotFoundHeading:       settings["site.notfound_heading"],
		NotFoundText:          settings["site.notfound_text"],
		NotFoundHtml:          settings["site.notfound_html"],
		HomePageMode:          homePageMode,
		HomeCustomHTML:        homeCustomHTML,
		AccountDisabledNotice: disabledNotice,
	}
	c.JSON(http.StatusOK, gin.H{"data": payload})
}

// siteSettingsUpdatePayload 仅用于 PATCH 部分更新：指针字段出现时才会写入对应配置，
// 未出现的字段保持原值，实现“只提交修改的字段”。
type siteSettingsUpdatePayload struct {
	SiteTitle             *string `json:"siteTitle"`
	ConsoleURL            *string `json:"consoleUrl"`
	SiteDescription       *string `json:"siteDescription"`
	SiteSlogan            *string `json:"siteSlogan"`
	SiteLogo              *string `json:"siteLogo"`
	About                 *string `json:"about"`
	AboutTitle            *string `json:"aboutTitle"`
	NotFoundMode          *string `json:"notFoundMode"`
	NotFoundHeading       *string `json:"notFoundHeading"`
	NotFoundText          *string `json:"notFoundText"`
	NotFoundHtml          *string `json:"notFoundHtml"`
	HomePageMode          *string `json:"homePageMode"`
	HomeCustomHTML        *string `json:"homeCustomHtml"`
	AccountDisabledNotice *string `json:"accountDisabledNotice"`
}

func (s *Server) handleAdminUpdateSiteSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload siteSettingsUpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	current, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	values := map[string]string{}
	setIf := func(configKey string, v *string) {
		if v != nil {
			values[configKey] = *v
		}
	}
	setIf("site.title", payload.SiteTitle)
	setIf("site.description", payload.SiteDescription)
	setIf("site.slogan", payload.SiteSlogan)
	setIf("site.logo", payload.SiteLogo)
	setIf("site.about", payload.About)
	setIf("site.about_title", payload.AboutTitle)
	setIf("site.notfound_mode", payload.NotFoundMode)
	setIf("site.notfound_heading", payload.NotFoundHeading)
	setIf("site.notfound_text", payload.NotFoundText)
	setIf("site.notfound_html", payload.NotFoundHtml)
	setIf("site.console_url", payload.ConsoleURL)

	if payload.AccountDisabledNotice != nil {
		notice := strings.TrimSpace(*payload.AccountDisabledNotice)
		if notice == "" {
			notice = defaultAccountDisabledNotice
		}
		values["account.disabled_notice"] = notice
	}

	if payload.HomePageMode != nil {
		mode := strings.TrimSpace(*payload.HomePageMode)
		if mode != "custom_html" {
			mode = "default"
		}
		values["site.home_page_mode"] = mode
		if mode == "custom_html" {
			if payload.HomeCustomHTML != nil {
				values["site.home_custom_html"] = *payload.HomeCustomHTML
			} else {
				values["site.home_custom_html"] = current["site.home_custom_html"]
			}
		} else {
			// 切回默认首页时清空自定义 HTML
			values["site.home_custom_html"] = ""
		}
	} else if payload.HomeCustomHTML != nil {
		values["site.home_custom_html"] = *payload.HomeCustomHTML
	}

	if len(values) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": "updated"})
		return
	}

	oldConsole := strings.TrimSpace(current["site.console_url"])
	newConsole := ""
	if consoleV, ok := values["site.console_url"]; ok {
		newConsole = strings.TrimSpace(consoleV)
	}

	if err := s.admin.UpdateSettings(c.Request.Context(), values); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// When console URL changes, rewrite stored thumbnail public URLs to the new domain.
	if newConsole != "" && !strings.EqualFold(strings.TrimRight(oldConsole, "/"), strings.TrimRight(newConsole, "/")) {
		if _, err := s.files.RewriteThumbnailPublicURLsToConsole(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "站点已保存，但缩略图链接更新失败: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": "updated"})
}

// siteLegalConfigKey 将 legal 类型映射到对应的配置 key。
func siteLegalConfigKey(legalType string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(legalType)) {
	case "terms":
		return "site.terms_of_service", nil
	case "privacy":
		return "site.privacy_policy", nil
	default:
		return "", errors.New("Invalid type")
	}
}

func (s *Server) handleAdminSiteLegal(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	configKey, err := siteLegalConfigKey(c.Param("type"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"content": settings[configKey]}})
}

func (s *Server) handleAdminUpdateSiteLegal(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	configKey, err := siteLegalConfigKey(c.Param("type"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var payload struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.admin.UpdateSettings(c.Request.Context(), map[string]string{configKey: payload.Content}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "updated"})
}

// ---------------------------------------------------------------------------
// General Settings (GET/PUT /admin/system/general)
// ---------------------------------------------------------------------------

// ConfigSidebarHidden 记录被隐藏的侧边栏 URL 路径（以逗号分隔，自然给所有用户读取）。
const ConfigSidebarHidden = "navigation.sidebar.hidden"

// allowedHiddenSidebarURLs 是后端可接受隐藏的侧边栏 URL 白名单，
// 对应前端 HIDEABLE_SIDEBAR_ITEMS 中非 critical 的项。关键页面与未列入白名单的路径一律拒绝，
// 防止绕过前端防御把任意路径写入隐藏配置。
var allowedHiddenSidebarURLs = map[string]struct{}{
	"/dashboard/shop":               {},
	"/dashboard/orders":             {},
	"/dashboard/tickets":            {},
	"/dashboard/notifications":      {},
	"/shop":                         {},
	"/dashboard/gallery":            {},
	"/dashboard/admin/audits":       {},
	"/dashboard/admin/redeem-codes": {},
	"/dashboard/admin/shop/products": {},
	"/dashboard/admin/shop/orders":   {},
	"/dashboard/admin/tickets":       {},
}

type generalSettingsPayload struct {
	ImageLoadRows                 int      `json:"imageLoadRows"`
	UserNotificationLimit         int      `json:"userNotificationLimit"`
	AdminImageDeleteDefaultReason string   `json:"adminImageDeleteDefaultReason"`
	SystemAutoDeleteDefaultReason string   `json:"systemAutoDeleteDefaultReason"`
	EnableCDN                     bool     `json:"enableCDN"`
	EnableGallery                 bool     `json:"enableGallery"`
	EnableHome                    bool     `json:"enableHome"`
	EnableApi                     bool     `json:"enableApi"`
	EnablePasskey                 bool     `json:"enablePasskey"`
	AllowRegistration             bool     `json:"allowRegistration"` // legacy, derived from registrationMode
	RegistrationMode              string   `json:"registrationMode"`  // open | oauth_only | closed
	HiddenSidebarItems            []string `json:"hiddenSidebarItems"`
}

// splitConfigList 将逗号分隔的配置字符串解析为去空项后的切片。
func splitConfigList(raw string) []string {
	out := []string{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func (s *Server) handleAdminGeneralSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	regMode := s.registrationMode(settings)
	payload := generalSettingsPayload{
		ImageLoadRows:                 normalizeImageLoadRows(settings["images.load_rows"]),
		UserNotificationLimit:         notifications.NormalizeRetentionLimit(settings[notifications.ConfigUserRetentionLimit]),
		AdminImageDeleteDefaultReason: notifications.NormalizeAdminDeleteReason(settings[notifications.ConfigAdminImageDeleteReason]),
		SystemAutoDeleteDefaultReason: notifications.NormalizeSystemAutoDeleteReason(settings[notifications.ConfigSystemAutoDeleteReason]),
		EnableCDN:                     settings["mail.cdn.enabled"] == "true",
		EnableGallery:                 settings["features.gallery"] != "false",
		EnableHome:                    settings["features.home"] != "false",
		EnableApi:                     settings["features.api"] != "false",
		EnablePasskey:                 settings["features.passkeys_enabled"] != "false",
		AllowRegistration:             regMode != "closed",
		RegistrationMode:              regMode,
		HiddenSidebarItems:            splitConfigList(settings[ConfigSidebarHidden]),
	}
	c.JSON(http.StatusOK, gin.H{"data": payload})
}

// generalSettingsUpdatePayload 用于 PATCH 部分更新：指针字段出现时才写入，未出现字段保持原值。
type generalSettingsUpdatePayload struct {
	ImageLoadRows                 *int     `json:"imageLoadRows"`
	UserNotificationLimit         *int     `json:"userNotificationLimit"`
	AdminImageDeleteDefaultReason *string  `json:"adminImageDeleteDefaultReason"`
	SystemAutoDeleteDefaultReason *string  `json:"systemAutoDeleteDefaultReason"`
	EnableCDN                     *bool    `json:"enableCDN"`
	EnableGallery                 *bool    `json:"enableGallery"`
	EnableHome                    *bool    `json:"enableHome"`
	EnableApi                     *bool    `json:"enableApi"`
	EnablePasskey                 *bool    `json:"enablePasskey"`
	AllowRegistration             *bool    `json:"allowRegistration"` // legacy, derived from registrationMode
	RegistrationMode              *string  `json:"registrationMode"`  // open | oauth_only | closed
	HiddenSidebarItems            []string `json:"hiddenSidebarItems"`
}

func (s *Server) handleAdminUpdateGeneralSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload generalSettingsUpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	values := map[string]string{}

	if payload.ImageLoadRows != nil {
		values["images.load_rows"] = strconv.Itoa(normalizeImageLoadRowsValue(*payload.ImageLoadRows))
	}
	if payload.UserNotificationLimit != nil {
		values[notifications.ConfigUserRetentionLimit] = strconv.Itoa(normalizeUserNotificationLimit(*payload.UserNotificationLimit))
	}
	if payload.AdminImageDeleteDefaultReason != nil {
		values[notifications.ConfigAdminImageDeleteReason] = notifications.NormalizeAdminDeleteReason(*payload.AdminImageDeleteDefaultReason)
	}
	if payload.SystemAutoDeleteDefaultReason != nil {
		values[notifications.ConfigSystemAutoDeleteReason] = notifications.NormalizeSystemAutoDeleteReason(*payload.SystemAutoDeleteDefaultReason)
	}
	if payload.EnableCDN != nil {
		values["mail.cdn.enabled"] = strconv.FormatBool(*payload.EnableCDN)
	}
	if payload.EnableGallery != nil {
		values["features.gallery"] = strconv.FormatBool(*payload.EnableGallery)
	}
	if payload.EnableHome != nil {
		values["features.home"] = strconv.FormatBool(*payload.EnableHome)
	}
	if payload.EnableApi != nil {
		values["features.api"] = strconv.FormatBool(*payload.EnableApi)
	}
	if payload.EnablePasskey != nil {
		values["features.passkeys_enabled"] = strconv.FormatBool(*payload.EnablePasskey)
	}

	// 注册模式：registrationMode 优先；否则回退到 allowRegistration（兼容旧请求）。
	if payload.RegistrationMode != nil {
		regMode := strings.ToLower(strings.TrimSpace(*payload.RegistrationMode))
		switch regMode {
		case "open", "oauth_only", "closed":
		default:
			regMode = "closed"
		}
		values["features.registration_mode"] = regMode
		values["features.allow_registration"] = strconv.FormatBool(regMode != "closed")
	} else if payload.AllowRegistration != nil {
		if *payload.AllowRegistration {
			values["features.registration_mode"] = "open"
		} else {
			values["features.registration_mode"] = "closed"
		}
		values["features.allow_registration"] = strconv.FormatBool(*payload.AllowRegistration)
	}

	if payload.HiddenSidebarItems != nil {
		hiddenItems := make([]string, 0, len(payload.HiddenSidebarItems))
		seen := make(map[string]struct{}, len(payload.HiddenSidebarItems))
		for _, item := range payload.HiddenSidebarItems {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := allowedHiddenSidebarURLs[item]; !ok {
				continue
			}
			if _, dup := seen[item]; dup {
				continue
			}
			seen[item] = struct{}{}
			hiddenItems = append(hiddenItems, item)
		}
		values[ConfigSidebarHidden] = strings.Join(hiddenItems, ",")
	}

	if len(values) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": "updated"})
		return
	}

	if err := s.admin.UpdateSettings(c.Request.Context(), values); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "updated"})
}

// ---------------------------------------------------------------------------
// Ticket Settings (GET/PUT /admin/system/tickets)
// ---------------------------------------------------------------------------

type ticketSettingsPayload struct {
	AttachmentStrategyID uint     `json:"attachmentStrategyId"`
	EmailNotifyEnabled   bool     `json:"emailNotifyEnabled"`
	EmailNotifyMode      string   `json:"emailNotifyMode"` // all_admins | selected
	EmailNotifyAdminIDs  []string `json:"emailNotifyAdminIds"`
}

func (s *Server) handleAdminTicketSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	mode := strings.ToLower(strings.TrimSpace(settings[tickets.ConfigEmailNotifyMode]))
	if mode != tickets.NotifyModeSelected {
		mode = tickets.NotifyModeAllAdmins
	}
	ids := []string{}
	for _, part := range strings.Split(settings[tickets.ConfigEmailNotifyAdminIDs], ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			ids = append(ids, part)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": ticketSettingsPayload{
		AttachmentStrategyID: parseUintSetting(settings[tickets.ConfigAttachmentStrategyID]),
		EmailNotifyEnabled:   settings[tickets.ConfigEmailNotifyEnabled] == "true",
		EmailNotifyMode:      mode,
		EmailNotifyAdminIDs:  ids,
	}})
}

// ticketSettingsUpdatePayload 用于 PATCH 部分更新：指针字段出现时才写入，未出现字段保持原值。
type ticketSettingsUpdatePayload struct {
	AttachmentStrategyID *uint    `json:"attachmentStrategyId"`
	EmailNotifyEnabled   *bool    `json:"emailNotifyEnabled"`
	EmailNotifyMode      *string  `json:"emailNotifyMode"` // all_admins | selected
	EmailNotifyAdminIDs  []string `json:"emailNotifyAdminIds"`
}

func (s *Server) handleAdminUpdateTicketSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload ticketSettingsUpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	values := map[string]string{}

	if payload.AttachmentStrategyID != nil {
		values[tickets.ConfigAttachmentStrategyID] = strconv.FormatUint(uint64(*payload.AttachmentStrategyID), 10)
	}
	if payload.EmailNotifyEnabled != nil {
		values[tickets.ConfigEmailNotifyEnabled] = strconv.FormatBool(*payload.EmailNotifyEnabled)
	}
	if payload.EmailNotifyMode != nil {
		mode := strings.ToLower(strings.TrimSpace(*payload.EmailNotifyMode))
		if mode != tickets.NotifyModeSelected {
			mode = tickets.NotifyModeAllAdmins
		}
		values[tickets.ConfigEmailNotifyMode] = mode
	}
	if payload.EmailNotifyAdminIDs != nil {
		ids := make([]string, 0, len(payload.EmailNotifyAdminIDs))
		for _, id := range payload.EmailNotifyAdminIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, err := strconv.ParseUint(id, 10, 64); err == nil {
				ids = append(ids, id)
			}
		}
		values[tickets.ConfigEmailNotifyAdminIDs] = strings.Join(ids, ",")
	}

	if len(values) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": "updated"})
		return
	}

	if err := s.admin.UpdateSettings(c.Request.Context(), values); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "updated"})
}

// ---------------------------------------------------------------------------
// Email Settings (GET/PUT /admin/system/email)
// ---------------------------------------------------------------------------

type emailSettingsPayload struct {
	SMTPHost                             string `json:"smtpHost"`
	SMTPPort                             string `json:"smtpPort"`
	SMTPUsername                         string `json:"smtpUsername"`
	SMTPPassword                         string `json:"smtpPassword"`
	SMTPFrom                             string `json:"smtpFrom"`
	SMTPSecure                           bool   `json:"smtpSecure"`
	MailTestSubject                      string `json:"mailTestSubject"`
	MailTestBody                         string `json:"mailTestBody"`
	MailRegisterVerifySubject            string `json:"mailRegisterVerifySubject"`
	MailRegisterVerifyBody               string `json:"mailRegisterVerifyBody"`
	MailRegisterSuccessSubject           string `json:"mailRegisterSuccessSubject"`
	MailRegisterSuccessBody              string `json:"mailRegisterSuccessBody"`
	MailLoginNotificationSubject         string `json:"mailLoginNotificationSubject"`
	MailLoginNotificationBody            string `json:"mailLoginNotificationBody"`
	MailForgotPasswordSubject            string `json:"mailForgotPasswordSubject"`
	MailForgotPasswordBody               string `json:"mailForgotPasswordBody"`
	MailTicketCreatedSubject             string `json:"mailTicketCreatedSubject"`
	MailTicketCreatedBody                string `json:"mailTicketCreatedBody"`
	MailTicketReplyUserSubject           string `json:"mailTicketReplyUserSubject"`
	MailTicketReplyUserBody              string `json:"mailTicketReplyUserBody"`
	MailTicketReplyAdminSubject          string `json:"mailTicketReplyAdminSubject"`
	MailTicketReplyAdminBody             string `json:"mailTicketReplyAdminBody"`
	MailTicketStatusSubject              string `json:"mailTicketStatusSubject"`
	MailTicketStatusBody                 string `json:"mailTicketStatusBody"`
	EnableRegisterVerify                 bool   `json:"enableRegisterVerify"`
	EnableLoginNotification              bool   `json:"enableLoginNotification"`
	EnableForgotPassword                 bool   `json:"enableForgotPassword"`
	EnableForgotPasswordTurnstile        bool   `json:"enableForgotPasswordTurnstile"`
	EnableForgotPasswordTurnstileRequest bool   `json:"enableForgotPasswordTurnstileRequest"`
	EnableForgotPasswordTurnstileReset   bool   `json:"enableForgotPasswordTurnstileReset"`
}

func (s *Server) handleAdminEmailSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	payload := emailSettingsPayload{
		SMTPHost:                             settings["mail.smtp.host"],
		SMTPPort:                             settings["mail.smtp.port"],
		SMTPUsername:                         settings["mail.smtp.username"],
		SMTPPassword:                         redactSecret(settings["mail.smtp.password"]),
		SMTPFrom:                             settings["mail.smtp.from"],
		SMTPSecure:                           settings["mail.smtp.secure"] == "true",
		MailTestSubject:                      settings["mail.template.test.subject"],
		MailTestBody:                         settings["mail.template.test.body"],
		MailRegisterVerifySubject:            settings["mail.template.register_verify.subject"],
		MailRegisterVerifyBody:               settings["mail.template.register_verify.body"],
		MailRegisterSuccessSubject:           settings["mail.template.register_success.subject"],
		MailRegisterSuccessBody:              settings["mail.template.register_success.body"],
		MailLoginNotificationSubject:         settings["mail.template.login_notification.subject"],
		MailLoginNotificationBody:            settings["mail.template.login_notification.body"],
		MailForgotPasswordSubject:            settings["mail.template.forgot_password.subject"],
		MailForgotPasswordBody:               settings["mail.template.forgot_password.body"],
		MailTicketCreatedSubject:             settings["mail.template.ticket_created.subject"],
		MailTicketCreatedBody:                settings["mail.template.ticket_created.body"],
		MailTicketReplyUserSubject:           settings["mail.template.ticket_reply_user.subject"],
		MailTicketReplyUserBody:              settings["mail.template.ticket_reply_user.body"],
		MailTicketReplyAdminSubject:          settings["mail.template.ticket_reply_admin.subject"],
		MailTicketReplyAdminBody:             settings["mail.template.ticket_reply_admin.body"],
		MailTicketStatusSubject:              settings["mail.template.ticket_status.subject"],
		MailTicketStatusBody:                 settings["mail.template.ticket_status.body"],
		EnableRegisterVerify:                 settings["mail.register.verify"] == "true",
		EnableLoginNotification:              settings["mail.login.notification"] == "true",
		EnableForgotPassword:                 settings["mail.forgot_password.enabled"] == "true",
		EnableForgotPasswordTurnstile:        settings["mail.forgot_password.turnstile"] == "true",
		EnableForgotPasswordTurnstileRequest: settings["mail.forgot_password.turnstile_request"] == "true",
		EnableForgotPasswordTurnstileReset:   settings["mail.forgot_password.turnstile_reset"] == "true",
	}
	c.JSON(http.StatusOK, gin.H{"data": payload})
}

// emailSettingsUpdatePayload 用于 PATCH 部分更新：指针字段出现时才写入，未出现字段保持原值。
type emailSettingsUpdatePayload struct {
	SMTPHost                             *string `json:"smtpHost"`
	SMTPPort                             *string `json:"smtpPort"`
	SMTPUsername                         *string `json:"smtpUsername"`
	SMTPPassword                         *string `json:"smtpPassword"`
	SMTPFrom                             *string `json:"smtpFrom"`
	SMTPSecure                           *bool   `json:"smtpSecure"`
	MailTestSubject                      *string `json:"mailTestSubject"`
	MailTestBody                         *string `json:"mailTestBody"`
	MailRegisterVerifySubject            *string `json:"mailRegisterVerifySubject"`
	MailRegisterVerifyBody               *string `json:"mailRegisterVerifyBody"`
	MailRegisterSuccessSubject           *string `json:"mailRegisterSuccessSubject"`
	MailRegisterSuccessBody              *string `json:"mailRegisterSuccessBody"`
	MailLoginNotificationSubject         *string `json:"mailLoginNotificationSubject"`
	MailLoginNotificationBody            *string `json:"mailLoginNotificationBody"`
	MailForgotPasswordSubject            *string `json:"mailForgotPasswordSubject"`
	MailForgotPasswordBody               *string `json:"mailForgotPasswordBody"`
	MailTicketCreatedSubject             *string `json:"mailTicketCreatedSubject"`
	MailTicketCreatedBody                *string `json:"mailTicketCreatedBody"`
	MailTicketReplyUserSubject           *string `json:"mailTicketReplyUserSubject"`
	MailTicketReplyUserBody              *string `json:"mailTicketReplyUserBody"`
	MailTicketReplyAdminSubject          *string `json:"mailTicketReplyAdminSubject"`
	MailTicketReplyAdminBody             *string `json:"mailTicketReplyAdminBody"`
	MailTicketStatusSubject              *string `json:"mailTicketStatusSubject"`
	MailTicketStatusBody                 *string `json:"mailTicketStatusBody"`
	EnableRegisterVerify                 *bool   `json:"enableRegisterVerify"`
	EnableLoginNotification              *bool   `json:"enableLoginNotification"`
	EnableForgotPassword                 *bool   `json:"enableForgotPassword"`
	EnableForgotPasswordTurnstile        *bool   `json:"enableForgotPasswordTurnstile"`
	EnableForgotPasswordTurnstileRequest *bool   `json:"enableForgotPasswordTurnstileRequest"`
	EnableForgotPasswordTurnstileReset   *bool   `json:"enableForgotPasswordTurnstileReset"`
}

func (s *Server) handleAdminUpdateEmailSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload emailSettingsUpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	values := map[string]string{}

	setIf := func(configKey string, v *string) {
		if v != nil {
			values[configKey] = *v
		}
	}
	setBoolIf := func(configKey string, v *bool) {
		if v != nil {
			values[configKey] = strconv.FormatBool(*v)
		}
	}

	setIf("mail.smtp.host", payload.SMTPHost)
	setIf("mail.smtp.port", payload.SMTPPort)
	setIf("mail.smtp.username", payload.SMTPUsername)
	setIf("mail.smtp.from", payload.SMTPFrom)
	setBoolIf("mail.smtp.secure", payload.SMTPSecure)

	// 密码为空或 "***"（未新增密钥）时保持原值。
	if payload.SMTPPassword != nil {
		smtpPassword := strings.TrimSpace(*payload.SMTPPassword)
		if smtpPassword == "" || smtpPassword == "***" {
			smtpPassword = settings["mail.smtp.password"]
		}
		values["mail.smtp.password"] = smtpPassword
	}

	setIf("mail.template.test.subject", payload.MailTestSubject)
	setIf("mail.template.test.body", payload.MailTestBody)
	setIf("mail.template.register_verify.subject", payload.MailRegisterVerifySubject)
	setIf("mail.template.register_verify.body", payload.MailRegisterVerifyBody)
	setIf("mail.template.register_success.subject", payload.MailRegisterSuccessSubject)
	setIf("mail.template.register_success.body", payload.MailRegisterSuccessBody)
	setIf("mail.template.login_notification.subject", payload.MailLoginNotificationSubject)
	setIf("mail.template.login_notification.body", payload.MailLoginNotificationBody)
	setIf("mail.template.forgot_password.subject", payload.MailForgotPasswordSubject)
	setIf("mail.template.forgot_password.body", payload.MailForgotPasswordBody)
	setIf("mail.template.ticket_created.subject", payload.MailTicketCreatedSubject)
	setIf("mail.template.ticket_created.body", payload.MailTicketCreatedBody)
	setIf("mail.template.ticket_reply_user.subject", payload.MailTicketReplyUserSubject)
	setIf("mail.template.ticket_reply_user.body", payload.MailTicketReplyUserBody)
	setIf("mail.template.ticket_reply_admin.subject", payload.MailTicketReplyAdminSubject)
	setIf("mail.template.ticket_reply_admin.body", payload.MailTicketReplyAdminBody)
	setIf("mail.template.ticket_status.subject", payload.MailTicketStatusSubject)
	setIf("mail.template.ticket_status.body", payload.MailTicketStatusBody)

	setBoolIf("mail.register.verify", payload.EnableRegisterVerify)
	setBoolIf("mail.login.notification", payload.EnableLoginNotification)
	setBoolIf("mail.forgot_password.enabled", payload.EnableForgotPassword)
	setBoolIf("mail.forgot_password.turnstile", payload.EnableForgotPasswordTurnstile)
	setBoolIf("mail.forgot_password.turnstile_request", payload.EnableForgotPasswordTurnstileRequest)
	setBoolIf("mail.forgot_password.turnstile_reset", payload.EnableForgotPasswordTurnstileReset)

	if len(values) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": "updated"})
		return
	}

	if err := s.admin.UpdateSettings(c.Request.Context(), values); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "updated"})
}

// ---------------------------------------------------------------------------
// Captcha Settings (GET/PUT /admin/system/captcha)
// ---------------------------------------------------------------------------

type captchaSettingsPayload struct {
	EnableCaptcha                      bool   `json:"enableCaptcha"`
	CaptchaProvider                    string `json:"captchaProvider"`
	CloudflareSiteKey                  string `json:"cloudflareSiteKey"`
	CloudflareSecretKey                string `json:"cloudflareSecretKey"`
	GeetestCaptchaID                   string `json:"geetestCaptchaId"`
	GeetestCaptchaKey                  string `json:"geetestCaptchaKey"`
	CapInstanceURL                     string `json:"capInstanceUrl"`
	CapSiteKey                         string `json:"capSiteKey"`
	CapSecretKey                       string `json:"capSecretKey"`
	EnableLoginCaptcha                 bool   `json:"enableLoginCaptcha"`
	EnableRegisterCaptcha              bool   `json:"enableRegisterCaptcha"`
	EnableRegisterVerifyCaptcha        bool   `json:"enableRegisterVerifyCaptcha"`
	EnableForgotPasswordRequestCaptcha bool   `json:"enableForgotPasswordRequestCaptcha"`
	EnableForgotPasswordResetCaptcha   bool   `json:"enableForgotPasswordResetCaptcha"`
	EnableRedeemCaptcha                bool   `json:"enableRedeemCaptcha"`
	EnableTicketCaptcha                bool   `json:"enableTicketCaptcha"`
}

type captchaSettingsResponse struct {
	captchaSettingsPayload
	CloudflareVerified       bool   `json:"cloudflareVerified"`
	CloudflareLastVerifiedAt string `json:"cloudflareLastVerifiedAt,omitempty"`
	GeetestVerified          bool   `json:"geetestVerified"`
	GeetestLastVerifiedAt    string `json:"geetestLastVerifiedAt,omitempty"`
	CapVerified              bool   `json:"capVerified"`
	CapLastVerifiedAt        string `json:"capLastVerifiedAt,omitempty"`
}

func (s *Server) handleAdminCaptchaSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := captchaSettingsResponse{
		captchaSettingsPayload: captchaSettingsPayload{
			EnableCaptcha:                      settings["captcha.enabled"] == "true",
			CaptchaProvider:                    settings["captcha.provider"],
			CloudflareSiteKey:                  settings["captcha.cloudflare.site_key"],
			CloudflareSecretKey:                redactSecret(settings["captcha.cloudflare.secret_key"]),
			GeetestCaptchaID:                   settings["captcha.geetest.captcha_id"],
			GeetestCaptchaKey:                  redactSecret(settings["captcha.geetest.captcha_key"]),
			CapInstanceURL:                     settings["captcha.cap.instance_url"],
			CapSiteKey:                         settings["captcha.cap.site_key"],
			CapSecretKey:                       redactSecret(settings["captcha.cap.secret_key"]),
			EnableLoginCaptcha:                 settings["captcha.login"] == "true",
			EnableRegisterCaptcha:              settings["captcha.register"] == "true",
			EnableRegisterVerifyCaptcha:        settings["captcha.register_verify"] == "true",
			EnableForgotPasswordRequestCaptcha: settings["captcha.forgot_password_request"] == "true",
			EnableForgotPasswordResetCaptcha:   settings["captcha.forgot_password_reset"] == "true",
			EnableRedeemCaptcha:                settings["captcha.redeem"] == "true",
			EnableTicketCaptcha:                settings["captcha.ticket"] == "true",
		},
		CloudflareLastVerifiedAt: settings["captcha.cloudflare.last_verified_at"],
		GeetestLastVerifiedAt:    settings["captcha.geetest.last_verified_at"],
		CapLastVerifiedAt:        settings["captcha.cap.last_verified_at"],
	}

	// 检查 Cloudflare 验证状态
	cloudflareExpectedSig := captcha.GenerateSignature(resp.CloudflareSiteKey, settings["captcha.cloudflare.secret_key"])
	cloudflareStoredSig := settings["captcha.cloudflare.last_verified_signature"]
	if cloudflareExpectedSig != "" && cloudflareStoredSig == cloudflareExpectedSig {
		resp.CloudflareVerified = true
	}
	// 检查 Geetest 验证状态
	geetestExpectedSig := captcha.GenerateGeetestSignature(resp.GeetestCaptchaID, settings["captcha.geetest.captcha_key"])
	geetestStoredSig := settings["captcha.geetest.last_verified_signature"]
	if geetestExpectedSig != "" && geetestStoredSig == geetestExpectedSig {
		resp.GeetestVerified = true
	}
	// 检查 Cap 验证状态
	capExpectedSig := captcha.GenerateCapSignature(resp.CapInstanceURL, resp.CapSiteKey, settings["captcha.cap.secret_key"])
	capStoredSig := settings["captcha.cap.last_verified_signature"]
	if capExpectedSig != "" && capStoredSig == capExpectedSig {
		resp.CapVerified = true
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}

// captchaSettingsUpdatePayload 用于 PATCH 部分更新：指针字段出现时才写入，未出现字段保持原值。
type captchaSettingsUpdatePayload struct {
	EnableCaptcha                      *bool   `json:"enableCaptcha"`
	CaptchaProvider                    *string `json:"captchaProvider"`
	CloudflareSiteKey                  *string `json:"cloudflareSiteKey"`
	CloudflareSecretKey                *string `json:"cloudflareSecretKey"`
	GeetestCaptchaID                   *string `json:"geetestCaptchaId"`
	GeetestCaptchaKey                  *string `json:"geetestCaptchaKey"`
	CapInstanceURL                     *string `json:"capInstanceUrl"`
	CapSiteKey                         *string `json:"capSiteKey"`
	CapSecretKey                       *string `json:"capSecretKey"`
	EnableLoginCaptcha                 *bool   `json:"enableLoginCaptcha"`
	EnableRegisterCaptcha              *bool   `json:"enableRegisterCaptcha"`
	EnableRegisterVerifyCaptcha        *bool   `json:"enableRegisterVerifyCaptcha"`
	EnableForgotPasswordRequestCaptcha *bool   `json:"enableForgotPasswordRequestCaptcha"`
	EnableForgotPasswordResetCaptcha   *bool   `json:"enableForgotPasswordResetCaptcha"`
	EnableRedeemCaptcha                *bool   `json:"enableRedeemCaptcha"`
	EnableTicketCaptcha                *bool   `json:"enableTicketCaptcha"`
}

func (s *Server) handleAdminUpdateCaptchaSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload captchaSettingsUpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	values := map[string]string{}

	// 解析各提供商密钥："***" 或空串（未新增密钥）时保持原值；未提供该字段则不处理。
	resolveSecret := func(current string, provided *string) (string, bool) {
		if provided == nil {
			return current, false
		}
		v := strings.TrimSpace(*provided)
		if v == "" || v == "***" {
			return current, true
		}
		return v, true
	}

	// Cloudflare
	currentCFSiteKey := settings["captcha.cloudflare.site_key"]
	currentCFSecret := settings["captcha.cloudflare.secret_key"]
	cfSiteKey := currentCFSiteKey
	if payload.CloudflareSiteKey != nil {
		cfSiteKey = *payload.CloudflareSiteKey
		values["captcha.cloudflare.site_key"] = cfSiteKey
	}
	cfSecret, cfSecretProvided := resolveSecret(currentCFSecret, payload.CloudflareSecretKey)
	if cfSecretProvided {
		values["captcha.cloudflare.secret_key"] = cfSecret
	}
	cfChanged := (payload.CloudflareSiteKey != nil && cfSiteKey != currentCFSiteKey) ||
		(payload.CloudflareSecretKey != nil && cfSecret != currentCFSecret)

	// Geetest
	currentGtID := settings["captcha.geetest.captcha_id"]
	currentGtKey := settings["captcha.geetest.captcha_key"]
	gtID := currentGtID
	if payload.GeetestCaptchaID != nil {
		gtID = *payload.GeetestCaptchaID
		values["captcha.geetest.captcha_id"] = gtID
	}
	gtKey, gtKeyProvided := resolveSecret(currentGtKey, payload.GeetestCaptchaKey)
	if gtKeyProvided {
		values["captcha.geetest.captcha_key"] = gtKey
	}
	gtChanged := (payload.GeetestCaptchaID != nil && gtID != currentGtID) ||
		(payload.GeetestCaptchaKey != nil && gtKey != currentGtKey)

	// Cap
	currentCapInstance := settings["captcha.cap.instance_url"]
	currentCapSiteKey := settings["captcha.cap.site_key"]
	currentCapSecret := settings["captcha.cap.secret_key"]
	capInstance := currentCapInstance
	if payload.CapInstanceURL != nil {
		capInstance = strings.TrimSpace(*payload.CapInstanceURL)
		if capInstance != "" {
			if normalized, err := captcha.NormalizeCapInstanceURL(capInstance); err == nil {
				capInstance = normalized
			}
		}
		values["captcha.cap.instance_url"] = capInstance
	}
	capSite := currentCapSiteKey
	if payload.CapSiteKey != nil {
		capSite = strings.TrimSpace(*payload.CapSiteKey)
		values["captcha.cap.site_key"] = capSite
	}
	capSecret, capSecretProvided := resolveSecret(currentCapSecret, payload.CapSecretKey)
	if capSecretProvided {
		values["captcha.cap.secret_key"] = capSecret
	}
	capChanged := (payload.CapInstanceURL != nil && capInstance != currentCapInstance) ||
		(payload.CapSiteKey != nil && capSite != currentCapSiteKey) ||
		(payload.CapSecretKey != nil && capSecret != currentCapSecret)

	// 场景开关
	setBoolIf := func(configKey string, v *bool) {
		if v != nil {
			values[configKey] = strconv.FormatBool(*v)
		}
	}
	if payload.EnableCaptcha != nil {
		values["captcha.enabled"] = strconv.FormatBool(*payload.EnableCaptcha)
	}
	if payload.CaptchaProvider != nil {
		values["captcha.provider"] = *payload.CaptchaProvider
	}
	setBoolIf("captcha.login", payload.EnableLoginCaptcha)
	setBoolIf("captcha.register", payload.EnableRegisterCaptcha)
	setBoolIf("captcha.register_verify", payload.EnableRegisterVerifyCaptcha)
	setBoolIf("captcha.forgot_password_request", payload.EnableForgotPasswordRequestCaptcha)
	setBoolIf("captcha.forgot_password_reset", payload.EnableForgotPasswordResetCaptcha)
	setBoolIf("captcha.redeem", payload.EnableRedeemCaptcha)
	setBoolIf("captcha.ticket", payload.EnableTicketCaptcha)

	// 当验证码启用且本次请求包含提供商相关变更时，检查所选提供商是否已验证
	providerFieldsProvided := payload.EnableCaptcha != nil || payload.CaptchaProvider != nil ||
		payload.CloudflareSiteKey != nil || payload.CloudflareSecretKey != nil ||
		payload.GeetestCaptchaID != nil || payload.GeetestCaptchaKey != nil ||
		payload.CapInstanceURL != nil || payload.CapSiteKey != nil || payload.CapSecretKey != nil
	if providerFieldsProvided {
		effectiveEnabled := settings["captcha.enabled"] == "true"
		if payload.EnableCaptcha != nil {
			effectiveEnabled = *payload.EnableCaptcha
		}
		if effectiveEnabled {
			provider := strings.TrimSpace(settings["captcha.provider"])
			if payload.CaptchaProvider != nil {
				provider = *payload.CaptchaProvider
			}
			if provider == "cloudflare" {
				if cfSiteKey == "" || cfSecret == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "使用 Cloudflare 验证时必须填写 Site Key 和 Secret Key"})
					return
				}
				newCFSig := captcha.GenerateSignature(cfSiteKey, cfSecret)
				if newCFSig == "" || settings["captcha.cloudflare.last_verified_signature"] != newCFSig {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cloudflare 配置已变更或未验证，请先点击测试并验证成功后再保存"})
					return
				}
			} else if provider == "geetest" {
				if gtID == "" || gtKey == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "使用极验验证时必须填写 Captcha ID 和 Captcha Key"})
					return
				}
				newGtSig := captcha.GenerateGeetestSignature(gtID, gtKey)
				if newGtSig == "" || settings["captcha.geetest.last_verified_signature"] != newGtSig {
					c.JSON(http.StatusBadRequest, gin.H{"error": "极验配置已变更或未验证，请先点击测试并验证成功后再保存"})
					return
				}
			} else if provider == "cap" {
				if capInstance == "" || capSite == "" || capSecret == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "使用 Cap 验证时必须填写 Instance URL、Site Key 和 Secret Key"})
					return
				}
				if _, err := captcha.NormalizeCapInstanceURL(capInstance); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				newCapSig := captcha.GenerateCapSignature(capInstance, capSite, capSecret)
				if newCapSig == "" || settings["captcha.cap.last_verified_signature"] != newCapSig {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cap 配置已变更或未验证，请先点击测试并验证成功后再保存"})
					return
				}
			}
		}
	}

	// 提供商配置变更时清除对应验证状态
	if cfChanged {
		values["captcha.cloudflare.last_verified_signature"] = ""
		values["captcha.cloudflare.last_verified_at"] = ""
	}
	if gtChanged {
		values["captcha.geetest.last_verified_signature"] = ""
		values["captcha.geetest.last_verified_at"] = ""
	}
	if capChanged {
		values["captcha.cap.last_verified_signature"] = ""
		values["captcha.cap.last_verified_at"] = ""
	}

	if len(values) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": "updated"})
		return
	}

	if err := s.admin.UpdateSettings(c.Request.Context(), values); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "updated"})
}

// ---------------------------------------------------------------------------
// OAuth Settings (GET/PUT /admin/system/oauth)
// ---------------------------------------------------------------------------

type oauthProviderSettings struct {
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Name         string `json:"name,omitempty"`
	AuthURL      string `json:"authUrl,omitempty"`
	TokenURL     string `json:"tokenUrl,omitempty"`
	UserInfoURL  string `json:"userInfoUrl,omitempty"`
	Scopes       string `json:"scopes,omitempty"`
}

type oauthSettingsPayload struct {
	Enabled         bool                  `json:"enabled"`
	AutoLinkByEmail bool                  `json:"autoLinkByEmail"`
	GitHub          oauthProviderSettings `json:"github"`
	Google          oauthProviderSettings `json:"google"`
	Discord         oauthProviderSettings `json:"discord"`
	Custom          oauthProviderSettings `json:"custom"`
}

func (s *Server) handleAdminOAuthSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	payload := oauthSettingsPayload{
		Enabled:         settings["oauth.enabled"] == "true",
		AutoLinkByEmail: settings["oauth.auto_link_by_email"] == "true",
		GitHub: oauthProviderSettings{
			Enabled:      settings["oauth.github.enabled"] == "true",
			ClientID:     settings["oauth.github.client_id"],
			ClientSecret: redactSecret(settings["oauth.github.client_secret"]),
		},
		Google: oauthProviderSettings{
			Enabled:      settings["oauth.google.enabled"] == "true",
			ClientID:     settings["oauth.google.client_id"],
			ClientSecret: redactSecret(settings["oauth.google.client_secret"]),
		},
		Discord: oauthProviderSettings{
			Enabled:      settings["oauth.discord.enabled"] == "true",
			ClientID:     settings["oauth.discord.client_id"],
			ClientSecret: redactSecret(settings["oauth.discord.client_secret"]),
		},
		Custom: oauthProviderSettings{
			Enabled:      settings["oauth.custom.enabled"] == "true",
			Name:         settings["oauth.custom.name"],
			ClientID:     settings["oauth.custom.client_id"],
			ClientSecret: redactSecret(settings["oauth.custom.client_secret"]),
			AuthURL:      settings["oauth.custom.auth_url"],
			TokenURL:     settings["oauth.custom.token_url"],
			UserInfoURL:  settings["oauth.custom.userinfo_url"],
			Scopes:       settings["oauth.custom.scopes"],
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": payload})
}

// oauthProviderUpdateSettings 用于 OAuth 提供商的部分更新：指针字段出现时才写入。
type oauthProviderUpdateSettings struct {
	Enabled      *bool   `json:"enabled"`
	ClientID     *string `json:"clientId"`
	ClientSecret *string `json:"clientSecret"`
	Name         *string `json:"name,omitempty"`
	AuthURL      *string `json:"authUrl,omitempty"`
	TokenURL     *string `json:"tokenUrl,omitempty"`
	UserInfoURL  *string `json:"userInfoUrl,omitempty"`
	Scopes       *string `json:"scopes,omitempty"`
}

type oauthSettingsUpdatePayload struct {
	Enabled         *bool                       `json:"enabled"`
	AutoLinkByEmail *bool                       `json:"autoLinkByEmail"`
	GitHub          *oauthProviderUpdateSettings `json:"github"`
	Google          *oauthProviderUpdateSettings `json:"google"`
	Discord         *oauthProviderUpdateSettings `json:"discord"`
	Custom          *oauthProviderUpdateSettings `json:"custom"`
}

func (s *Server) handleAdminUpdateOAuthSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload oauthSettingsUpdatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	values := map[string]string{}

	// 密钥为空或 "***"（未新增密钥）时保持原值；未提供该字段则不处理。
	resolveSecret := func(current string, provided *string) string {
		if provided == nil {
			return current
		}
		v := strings.TrimSpace(*provided)
		if v == "" || v == "***" {
			return current
		}
		return v
	}

	if payload.Enabled != nil {
		values["oauth.enabled"] = strconv.FormatBool(*payload.Enabled)
	}
	if payload.AutoLinkByEmail != nil {
		values["oauth.auto_link_by_email"] = strconv.FormatBool(*payload.AutoLinkByEmail)
	}

	writeProvider := func(prefix string, p *oauthProviderUpdateSettings) {
		if p == nil {
			return
		}
		if p.Enabled != nil {
			values[prefix+".enabled"] = strconv.FormatBool(*p.Enabled)
		}
		if p.ClientID != nil {
			values[prefix+".client_id"] = strings.TrimSpace(*p.ClientID)
		}
		if p.ClientSecret != nil {
			values[prefix+".client_secret"] = resolveSecret(settings[prefix+".client_secret"], p.ClientSecret)
		}
		if p.Name != nil {
			values[prefix+".name"] = strings.TrimSpace(*p.Name)
		}
		if p.AuthURL != nil {
			values[prefix+".auth_url"] = strings.TrimRight(strings.TrimSpace(*p.AuthURL), "?&")
		}
		if p.TokenURL != nil {
			values[prefix+".token_url"] = strings.TrimRight(strings.TrimSpace(*p.TokenURL), "?&")
		}
		if p.UserInfoURL != nil {
			values[prefix+".userinfo_url"] = strings.TrimRight(strings.TrimSpace(*p.UserInfoURL), "?&")
		}
		if p.Scopes != nil {
			values[prefix+".scopes"] = firstNonEmptyTrim(*p.Scopes, "openid profile email")
		}
	}
	writeProvider("oauth.github", payload.GitHub)
	writeProvider("oauth.google", payload.Google)
	writeProvider("oauth.discord", payload.Discord)
	writeProvider("oauth.custom", payload.Custom)

	// 自定义 OAuth 校验：仅在生效状态为启用且本次请求包含 custom 相关字段时执行。
	if payload.Custom != nil {
		effectiveEnabled := settings["oauth.enabled"] == "true"
		if payload.Enabled != nil {
			effectiveEnabled = *payload.Enabled
		}
		effectiveCustomEnabled := settings["oauth.custom.enabled"] == "true"
		if payload.Custom.Enabled != nil {
			effectiveCustomEnabled = *payload.Custom.Enabled
		}

		if effectiveEnabled && effectiveCustomEnabled {
			clientID := settings["oauth.custom.client_id"]
			if payload.Custom.ClientID != nil {
				clientID = *payload.Custom.ClientID
			}
			customSecret := resolveSecret(settings["oauth.custom.client_secret"], payload.Custom.ClientSecret)
			customAuthURL := settings["oauth.custom.auth_url"]
			if payload.Custom.AuthURL != nil {
				customAuthURL = *payload.Custom.AuthURL
			}
			customTokenURL := settings["oauth.custom.token_url"]
			if payload.Custom.TokenURL != nil {
				customTokenURL = *payload.Custom.TokenURL
			}
			customUserInfoURL := settings["oauth.custom.userinfo_url"]
			if payload.Custom.UserInfoURL != nil {
				customUserInfoURL = *payload.Custom.UserInfoURL
			}

			if strings.TrimSpace(clientID) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "自定义 OAuth 已启用，请填写 Client ID"})
				return
			}
			if customSecret == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "自定义 OAuth 已启用，请填写 Client Secret"})
				return
			}
			for _, item := range []struct {
				label string
				value string
			}{
				{"Auth URL", customAuthURL},
				{"Token URL", customTokenURL},
				{"UserInfo URL", customUserInfoURL},
			} {
				u, err := url.Parse(strings.TrimSpace(item.value))
				if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": item.label + " 必须是完整 http(s) 地址，例如 https://casdoor.example.com/login/oauth/authorize"})
					return
				}
				host := strings.ToLower(u.Hostname())
				if host == "localhost" || strings.HasSuffix(host, ".localhost") || host == "127.0.0.1" || host == "::1" || host == "metadata.google.internal" {
					c.JSON(http.StatusBadRequest, gin.H{"error": item.label + " 不能指向本机或元数据地址"})
					return
				}
			}
		}
	}

	if len(values) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": "updated"})
		return
	}

	if err := s.admin.UpdateSettings(c.Request.Context(), values); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "updated"})
}

func firstNonEmptyTrim(values ...string) string {
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}
