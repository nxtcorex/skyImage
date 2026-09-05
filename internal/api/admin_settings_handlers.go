package api

import (
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
// Site Settings (GET/PUT /admin/system/site)
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
	TermsOfService        string `json:"termsOfService"`
	PrivacyPolicy         string `json:"privacyPolicy"`
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
		TermsOfService:        settings["site.terms_of_service"],
		PrivacyPolicy:         settings["site.privacy_policy"],
		HomePageMode:          homePageMode,
		HomeCustomHTML:        homeCustomHTML,
		AccountDisabledNotice: disabledNotice,
	}
	c.JSON(http.StatusOK, gin.H{"data": payload})
}

func (s *Server) handleAdminUpdateSiteSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload siteSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notice := strings.TrimSpace(payload.AccountDisabledNotice)
	if notice == "" {
		notice = defaultAccountDisabledNotice
	}
	homePageMode := strings.TrimSpace(payload.HomePageMode)
	if homePageMode != "custom_html" {
		homePageMode = "default"
	}
	homeCustomHTML := ""
	if homePageMode == "custom_html" {
		homeCustomHTML = payload.HomeCustomHTML
	}

	values := map[string]string{
		"site.title":              payload.SiteTitle,
		"site.console_url":        payload.ConsoleURL,
		"site.description":        payload.SiteDescription,
		"site.slogan":             payload.SiteSlogan,
		"site.logo":               payload.SiteLogo,
		"site.about":              payload.About,
		"site.about_title":        payload.AboutTitle,
		"site.notfound_mode":      payload.NotFoundMode,
		"site.notfound_heading":   payload.NotFoundHeading,
		"site.notfound_text":      payload.NotFoundText,
		"site.notfound_html":      payload.NotFoundHtml,
		"site.terms_of_service":   payload.TermsOfService,
		"site.privacy_policy":     payload.PrivacyPolicy,
		"site.home_page_mode":     homePageMode,
		"site.home_custom_html":   homeCustomHTML,
		"account.disabled_notice": notice,
	}

	oldSettings, _ := s.admin.GetSettings(c.Request.Context())
	oldConsole := strings.TrimSpace(oldSettings["site.console_url"])
	newConsole := strings.TrimSpace(payload.ConsoleURL)

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

func (s *Server) handleAdminUpdateGeneralSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload generalSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	adminDeleteReason := notifications.NormalizeAdminDeleteReason(payload.AdminImageDeleteDefaultReason)
	systemAutoDeleteReason := notifications.NormalizeSystemAutoDeleteReason(payload.SystemAutoDeleteDefaultReason)

	regMode := strings.ToLower(strings.TrimSpace(payload.RegistrationMode))
	switch regMode {
	case "open", "oauth_only", "closed":
	default:
		// Backward compatible: allowRegistration bool only
		if payload.AllowRegistration {
			regMode = "open"
		} else {
			regMode = "closed"
		}
	}

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

	values := map[string]string{
		"images.load_rows":                         strconv.Itoa(normalizeImageLoadRowsValue(payload.ImageLoadRows)),
		notifications.ConfigUserRetentionLimit:     strconv.Itoa(normalizeUserNotificationLimit(payload.UserNotificationLimit)),
		notifications.ConfigAdminImageDeleteReason: adminDeleteReason,
		notifications.ConfigSystemAutoDeleteReason: systemAutoDeleteReason,
		"mail.cdn.enabled":                         strconv.FormatBool(payload.EnableCDN),
		"features.gallery":                         strconv.FormatBool(payload.EnableGallery),
		"features.home":                            strconv.FormatBool(payload.EnableHome),
		"features.api":                             strconv.FormatBool(payload.EnableApi),
		"features.passkeys_enabled":                strconv.FormatBool(payload.EnablePasskey),
		"features.registration_mode":               regMode,
		"features.allow_registration":              strconv.FormatBool(regMode != "closed"),
		ConfigSidebarHidden:                        strings.Join(hiddenItems, ","),
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

func (s *Server) handleAdminUpdateTicketSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload ticketSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mode := strings.ToLower(strings.TrimSpace(payload.EmailNotifyMode))
	if mode != tickets.NotifyModeSelected {
		mode = tickets.NotifyModeAllAdmins
	}
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
	values := map[string]string{
		tickets.ConfigAttachmentStrategyID: strconv.FormatUint(uint64(payload.AttachmentStrategyID), 10),
		tickets.ConfigEmailNotifyEnabled:   strconv.FormatBool(payload.EmailNotifyEnabled),
		tickets.ConfigEmailNotifyMode:      mode,
		tickets.ConfigEmailNotifyAdminIDs:  strings.Join(ids, ","),
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

func (s *Server) handleAdminUpdateEmailSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload emailSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	smtpPassword := strings.TrimSpace(payload.SMTPPassword)
	if smtpPassword == "" || smtpPassword == "***" {
		smtpPassword = settings["mail.smtp.password"]
	}

	values := map[string]string{
		"mail.smtp.host":                           payload.SMTPHost,
		"mail.smtp.port":                           payload.SMTPPort,
		"mail.smtp.username":                       payload.SMTPUsername,
		"mail.smtp.password":                       smtpPassword,
		"mail.smtp.from":                           payload.SMTPFrom,
		"mail.smtp.secure":                         strconv.FormatBool(payload.SMTPSecure),
		"mail.template.test.subject":               payload.MailTestSubject,
		"mail.template.test.body":                  payload.MailTestBody,
		"mail.template.register_verify.subject":    payload.MailRegisterVerifySubject,
		"mail.template.register_verify.body":       payload.MailRegisterVerifyBody,
		"mail.template.register_success.subject":   payload.MailRegisterSuccessSubject,
		"mail.template.register_success.body":      payload.MailRegisterSuccessBody,
		"mail.template.login_notification.subject": payload.MailLoginNotificationSubject,
		"mail.template.login_notification.body":    payload.MailLoginNotificationBody,
		"mail.template.forgot_password.subject":    payload.MailForgotPasswordSubject,
		"mail.template.forgot_password.body":       payload.MailForgotPasswordBody,
		"mail.template.ticket_created.subject":     payload.MailTicketCreatedSubject,
		"mail.template.ticket_created.body":        payload.MailTicketCreatedBody,
		"mail.template.ticket_reply_user.subject":  payload.MailTicketReplyUserSubject,
		"mail.template.ticket_reply_user.body":     payload.MailTicketReplyUserBody,
		"mail.template.ticket_reply_admin.subject": payload.MailTicketReplyAdminSubject,
		"mail.template.ticket_reply_admin.body":    payload.MailTicketReplyAdminBody,
		"mail.template.ticket_status.subject":      payload.MailTicketStatusSubject,
		"mail.template.ticket_status.body":         payload.MailTicketStatusBody,
		"mail.register.verify":                     strconv.FormatBool(payload.EnableRegisterVerify),
		"mail.login.notification":                  strconv.FormatBool(payload.EnableLoginNotification),
		"mail.forgot_password.enabled":             strconv.FormatBool(payload.EnableForgotPassword),
		"mail.forgot_password.turnstile":           strconv.FormatBool(payload.EnableForgotPasswordTurnstile),
		"mail.forgot_password.turnstile_request":   strconv.FormatBool(payload.EnableForgotPasswordTurnstileRequest),
		"mail.forgot_password.turnstile_reset":     strconv.FormatBool(payload.EnableForgotPasswordTurnstileReset),
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

func (s *Server) handleAdminUpdateCaptchaSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload captchaSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 处理密钥 "***" 保留逻辑
	cloudflareSecretKey := strings.TrimSpace(payload.CloudflareSecretKey)
	if cloudflareSecretKey == "" || cloudflareSecretKey == "***" {
		cloudflareSecretKey = settings["captcha.cloudflare.secret_key"]
	}
	geetestCaptchaKey := strings.TrimSpace(payload.GeetestCaptchaKey)
	if geetestCaptchaKey == "" || geetestCaptchaKey == "***" {
		geetestCaptchaKey = settings["captcha.geetest.captcha_key"]
	}
	capSecretKey := strings.TrimSpace(payload.CapSecretKey)
	if capSecretKey == "" || capSecretKey == "***" {
		capSecretKey = settings["captcha.cap.secret_key"]
	}
	capInstanceURL := strings.TrimSpace(payload.CapInstanceURL)
	if capInstanceURL != "" {
		if normalized, err := captcha.NormalizeCapInstanceURL(capInstanceURL); err == nil {
			capInstanceURL = normalized
		}
	}
	capSiteKey := strings.TrimSpace(payload.CapSiteKey)

	// 检测配置变更
	currentCloudflareSiteKey := settings["captcha.cloudflare.site_key"]
	currentCloudflareSecretKey := settings["captcha.cloudflare.secret_key"]
	currentGeetestCaptchaID := settings["captcha.geetest.captcha_id"]
	currentGeetestCaptchaKey := settings["captcha.geetest.captcha_key"]
	currentCapInstanceURL := settings["captcha.cap.instance_url"]
	currentCapSiteKey := settings["captcha.cap.site_key"]
	currentCapSecretKey := settings["captcha.cap.secret_key"]

	cloudflareConfigChanged := payload.CloudflareSiteKey != currentCloudflareSiteKey || cloudflareSecretKey != currentCloudflareSecretKey
	geetestConfigChanged := payload.GeetestCaptchaID != currentGeetestCaptchaID || geetestCaptchaKey != currentGeetestCaptchaKey
	capConfigChanged := capInstanceURL != currentCapInstanceURL || capSiteKey != currentCapSiteKey || capSecretKey != currentCapSecretKey

	// 当验证码启用时，检查所选提供商是否已验证
	if payload.EnableCaptcha {
		if payload.CaptchaProvider == "cloudflare" {
			if payload.CloudflareSiteKey == "" || cloudflareSecretKey == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "使用 Cloudflare 验证时必须填写 Site Key 和 Secret Key"})
				return
			}
			newCloudflareSig := captcha.GenerateSignature(payload.CloudflareSiteKey, cloudflareSecretKey)
			if newCloudflareSig == "" || settings["captcha.cloudflare.last_verified_signature"] != newCloudflareSig {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cloudflare 配置已变更或未验证，请先点击测试并验证成功后再保存"})
				return
			}
		} else if payload.CaptchaProvider == "geetest" {
			if payload.GeetestCaptchaID == "" || geetestCaptchaKey == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "使用极验验证时必须填写 Captcha ID 和 Captcha Key"})
				return
			}
			newGeetestSig := captcha.GenerateGeetestSignature(payload.GeetestCaptchaID, geetestCaptchaKey)
			if newGeetestSig == "" || settings["captcha.geetest.last_verified_signature"] != newGeetestSig {
				c.JSON(http.StatusBadRequest, gin.H{"error": "极验配置已变更或未验证，请先点击测试并验证成功后再保存"})
				return
			}
		} else if payload.CaptchaProvider == "cap" {
			if capInstanceURL == "" || capSiteKey == "" || capSecretKey == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "使用 Cap 验证时必须填写 Instance URL、Site Key 和 Secret Key"})
				return
			}
			if _, err := captcha.NormalizeCapInstanceURL(capInstanceURL); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			newCapSig := captcha.GenerateCapSignature(capInstanceURL, capSiteKey, capSecretKey)
			if newCapSig == "" || settings["captcha.cap.last_verified_signature"] != newCapSig {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cap 配置已变更或未验证，请先点击测试并验证成功后再保存"})
				return
			}
		}
	}

	values := map[string]string{
		"captcha.enabled":                 strconv.FormatBool(payload.EnableCaptcha),
		"captcha.provider":                payload.CaptchaProvider,
		"captcha.cloudflare.site_key":     payload.CloudflareSiteKey,
		"captcha.cloudflare.secret_key":   cloudflareSecretKey,
		"captcha.geetest.captcha_id":      payload.GeetestCaptchaID,
		"captcha.geetest.captcha_key":     geetestCaptchaKey,
		"captcha.cap.instance_url":        capInstanceURL,
		"captcha.cap.site_key":            capSiteKey,
		"captcha.cap.secret_key":          capSecretKey,
		"captcha.login":                   strconv.FormatBool(payload.EnableLoginCaptcha),
		"captcha.register":                strconv.FormatBool(payload.EnableRegisterCaptcha),
		"captcha.register_verify":         strconv.FormatBool(payload.EnableRegisterVerifyCaptcha),
		"captcha.forgot_password_request": strconv.FormatBool(payload.EnableForgotPasswordRequestCaptcha),
		"captcha.forgot_password_reset":   strconv.FormatBool(payload.EnableForgotPasswordResetCaptcha),
		"captcha.redeem":                  strconv.FormatBool(payload.EnableRedeemCaptcha),
		"captcha.ticket":                  strconv.FormatBool(payload.EnableTicketCaptcha),
	}

	// Cloudflare 配置变更时清除验证状态
	if cloudflareConfigChanged {
		values["captcha.cloudflare.last_verified_signature"] = ""
		values["captcha.cloudflare.last_verified_at"] = ""
	}
	// Geetest 配置变更时清除验证状态
	if geetestConfigChanged {
		values["captcha.geetest.last_verified_signature"] = ""
		values["captcha.geetest.last_verified_at"] = ""
	}
	// Cap 配置变更时清除验证状态
	if capConfigChanged {
		values["captcha.cap.last_verified_signature"] = ""
		values["captcha.cap.last_verified_at"] = ""
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

func (s *Server) handleAdminUpdateOAuthSettings(c *gin.Context) {
	if !requireSuperAdmin(c) {
		return
	}
	var payload oauthSettingsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	settings, _ := s.admin.GetSettings(c.Request.Context())

	githubSecret := strings.TrimSpace(payload.GitHub.ClientSecret)
	if githubSecret == "" || githubSecret == "***" {
		githubSecret = settings["oauth.github.client_secret"]
	}
	googleSecret := strings.TrimSpace(payload.Google.ClientSecret)
	if googleSecret == "" || googleSecret == "***" {
		googleSecret = settings["oauth.google.client_secret"]
	}
	discordSecret := strings.TrimSpace(payload.Discord.ClientSecret)
	if discordSecret == "" || discordSecret == "***" {
		discordSecret = settings["oauth.discord.client_secret"]
	}
	customSecret := strings.TrimSpace(payload.Custom.ClientSecret)
	if customSecret == "" || customSecret == "***" {
		customSecret = settings["oauth.custom.client_secret"]
	}

	if payload.Enabled && payload.Custom.Enabled {
		if strings.TrimSpace(payload.Custom.ClientID) == "" {
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
			{"Auth URL", payload.Custom.AuthURL},
			{"Token URL", payload.Custom.TokenURL},
			{"UserInfo URL", payload.Custom.UserInfoURL},
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

	values := map[string]string{
		"oauth.enabled":               strconv.FormatBool(payload.Enabled),
		"oauth.auto_link_by_email":    strconv.FormatBool(payload.AutoLinkByEmail),
		"oauth.github.enabled":        strconv.FormatBool(payload.GitHub.Enabled),
		"oauth.github.client_id":      strings.TrimSpace(payload.GitHub.ClientID),
		"oauth.github.client_secret":  githubSecret,
		"oauth.google.enabled":        strconv.FormatBool(payload.Google.Enabled),
		"oauth.google.client_id":      strings.TrimSpace(payload.Google.ClientID),
		"oauth.google.client_secret":  googleSecret,
		"oauth.discord.enabled":       strconv.FormatBool(payload.Discord.Enabled),
		"oauth.discord.client_id":     strings.TrimSpace(payload.Discord.ClientID),
		"oauth.discord.client_secret": discordSecret,
		"oauth.custom.enabled":        strconv.FormatBool(payload.Custom.Enabled),
		"oauth.custom.name":           strings.TrimSpace(payload.Custom.Name),
		"oauth.custom.client_id":      strings.TrimSpace(payload.Custom.ClientID),
		"oauth.custom.client_secret":  customSecret,
		"oauth.custom.auth_url":       strings.TrimRight(strings.TrimSpace(payload.Custom.AuthURL), "?&"),
		"oauth.custom.token_url":      strings.TrimRight(strings.TrimSpace(payload.Custom.TokenURL), "?&"),
		"oauth.custom.userinfo_url":   strings.TrimRight(strings.TrimSpace(payload.Custom.UserInfoURL), "?&"),
		"oauth.custom.scopes":         firstNonEmptyTrim(payload.Custom.Scopes, "openid profile email"),
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
