package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	stdhtml "html"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"skyimage/internal/admin"
	"skyimage/internal/captcha"
	"skyimage/internal/config"
	"skyimage/internal/data"
	"skyimage/internal/files"
	"skyimage/internal/installer"
	"skyimage/internal/mail"
	"skyimage/internal/middleware"
	"skyimage/internal/notifications"
	"skyimage/internal/oauth"
	"skyimage/internal/passkey"
	"skyimage/internal/redeem"
	"skyimage/internal/session"
	"skyimage/internal/shop"
	"skyimage/internal/tickets"
	"skyimage/internal/users"
	"skyimage/internal/verification"
)

type Server struct {
	engine        *gin.Engine
	mu            sync.RWMutex
	cfg           config.Config
	db            *gorm.DB
	installer     *installer.Service
	admin         *admin.Service
	files         *files.Service
	users         *users.Service
	notifications *notifications.Service
	redeem        *redeem.Service
	shop          *shop.Service
	tickets       *tickets.Service
	mail          *mail.Service
	captcha       *captcha.Service
	verification  *verification.Service
	oauth         *oauth.Service
	passkeys      *passkey.Service
	session       *session.Manager
	authLimiter   *requestLimiter
	publicPaths   map[string]struct{}
	stopShopExp   chan struct{}
}

func NewServer(cfg config.Config, db *gorm.DB) *Server {
	// Default to release. Set GIN_MODE=debug for local Vite LAN proxy logging/tools.
	// Docker/release images set GIN_MODE=release explicitly.
	switch strings.ToLower(strings.TrimSpace(os.Getenv("GIN_MODE"))) {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	trustedProxies := cfg.TrustedProxies
	if len(trustedProxies) == 0 {
		trustedProxies = nil
	}
	if err := engine.SetTrustedProxies(trustedProxies); err != nil {
		log.Printf("set trusted proxies failed: %v", err)
	}

	// 构建 CORS 允许的源列表
	allowedOrigins := []string{cfg.PublicBaseURL}
	if len(cfg.CORSAllowedOrigins) > 0 {
		allowedOrigins = append(allowedOrigins, cfg.CORSAllowedOrigins...)
	}

	engine.Use(
		gin.Logger(),
		gin.Recovery(),
		middleware.CORS(allowedOrigins...),
	)
	engine.RedirectTrailingSlash = false

	s := &Server{
		engine:      engine,
		authLimiter: newRequestLimiter(),
		publicPaths: make(map[string]struct{}),
	}
	s.applyRuntimeConfig(cfg, db)
	s.installer = installer.New(db, cfg, s.applyRuntimeConfig)
	s.registerRoutes()
	return s
}

func (s *Server) applyRuntimeConfig(cfg config.Config, db *gorm.DB) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db != nil && s.db != db {
		if sqlDB, err := s.db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
	s.cfg = cfg
	s.db = db
	adminService := admin.New(db)
	s.admin = adminService
	s.files = files.New(db, cfg)
	s.users = users.New(db)
	s.notifications = notifications.New(db)
	s.redeem = redeem.New(db)
	if s.shop == nil {
		s.shop = shop.New(db, adminService)
	} else {
		s.shop.SetDB(db)
		s.shop.SetAdmin(adminService)
	}
	s.mail = mail.New(adminService)
	if s.tickets == nil {
		s.tickets = tickets.New(db, s.files, s.notifications)
	} else {
		s.tickets.SetDB(db)
		s.tickets.SetFiles(s.files)
		s.tickets.SetNotifications(s.notifications)
	}
	s.tickets.SetMail(s.mail)
	s.captcha = captcha.New(adminService)
	s.verification = verification.New()
	if s.oauth == nil {
		s.oauth = oauth.New(db, adminService)
	} else {
		s.oauth.SetDB(db)
		s.oauth.SetSettings(adminService)
	}
	if s.passkeys == nil {
		s.passkeys = passkey.New(db, adminService, cfg.PublicBaseURL)
	} else {
		s.passkeys.SetDB(db)
		s.passkeys.SetSettings(adminService)
	}
	if s.session == nil {
		s.session = session.NewManager(db, 24*time.Hour)
	} else {
		s.session.SetDB(db)
	}
	if s.installer != nil {
		s.installer.SetRuntime(db, cfg)
	}
	s.ensureShopExpiryLoop()
}

func (s *Server) Run(ctx context.Context) error {
	// 演示站模式：自动初始化
	if s.cfg.DemoMode && s.cfg.SkipInstall {
		log.Println("演示站模式：开始自动初始化...")
		if err := s.autoInitialize(ctx); err != nil {
			log.Printf("演示站自动初始化失败: %v", err)
		} else {
			log.Println("演示站自动初始化完成")
		}
	}

	// 未安装状态下提示安装向导密码来源：环境变量固定值或本次启动随机生成的密码。
	if status, err := s.installer.Status(ctx); err == nil && !status.Installed {
		if password, fromEnv := s.installer.InstallPassword(); fromEnv {
			log.Println("安装向导已启用密码验证（密码来源：INSTALL_PASSWORD 环境变量）")
		} else {
			log.Printf("未设置 INSTALL_PASSWORD，已自动生成本次启动的安装向导密码：%s", password)
			log.Println("提示：该密码每次重启都会重新生成，也可通过环境变量 INSTALL_PASSWORD 固定安装密码")
		}
	}

	srv := &http.Server{
		Addr:    s.cfg.HTTPAddr,
		Handler: s.engine,
	}
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctxShutdown)
	case err := <-errCh:
		return err
	}
}

// autoInitialize 演示站自动初始化
func (s *Server) autoInitialize(ctx context.Context) error {
	// 检查是否已初始化
	status, err := s.installer.Status(ctx)
	if err != nil {
		return fmt.Errorf("检查初始化状态失败: %w", err)
	}

	if status.Installed {
		log.Println("演示站已初始化，跳过自动初始化")
		return nil
	}

	// 准备初始化输入
	input := installer.RunInput{
		DatabaseType:  "sqlite",
		DatabasePath:  s.cfg.DatabasePath,
		SiteName:      s.cfg.SiteName,
		AdminName:     s.cfg.AdminUsername,
		AdminEmail:    s.cfg.AdminEmail,
		AdminPassword: s.cfg.AdminPassword,
	}

	// 运行初始化
	_, err = s.installer.Run(ctx, input)
	if err != nil {
		return fmt.Errorf("运行初始化失败: %w", err)
	}

	// 创建演示站普通用户
	if err := s.ensureDemoUser(ctx); err != nil {
		log.Printf("创建演示站普通用户失败: %v", err)
	}

	return nil
}

// ensureDemoUser 创建演示站默认普通用户（如果不存在）
func (s *Server) ensureDemoUser(ctx context.Context) error {
	email := strings.ToLower(strings.TrimSpace(s.cfg.DemoUserEmail))
	if email == "" {
		return nil
	}

	// 检查用户是否已存在
	var count int64
	if err := s.db.WithContext(ctx).Model(&data.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(s.cfg.DemoUserPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// 获取默认用户组
	var group data.Group
	if err := s.db.WithContext(ctx).Where("is_default = ?", true).First(&group).Error; err != nil {
		return fmt.Errorf("查找默认用户组: %w", err)
	}

	user := data.User{
		GroupID:      &group.ID,
		Name:         s.cfg.DemoUserUsername,
		Email:        email,
		PasswordHash: string(hashed),
		IsAdmin:      false,
		Status:       1,
		Configs:      datatypes.JSON([]byte(`{"default_visibility":"private"}`)),
	}
	if err := users.CreateUserWithGeneratedID(s.db.WithContext(ctx), &user); err != nil {
		return fmt.Errorf("创建演示用户: %w", err)
	}

	log.Printf("演示站普通用户已创建: %s (%s)", user.Name, email)
	return nil
}

func (s *Server) ensureShopExpiryLoop() {
	if s.stopShopExp != nil {
		return
	}
	s.stopShopExp = make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopShopExp:
				return
			case <-ticker.C:
				if s.shop == nil {
					continue
				}
				if _, err := s.shop.ExpireDueMemberships(context.Background(), 200); err != nil {
					log.Printf("shop membership expiry: %v", err)
				}
			}
		}
	}()
}

func (s *Server) healthHandler(c *gin.Context) {
	status, err := s.installer.Status(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"status":    "ok",
			"installer": status,
		},
	})
}

func (s *Server) robotsHandler(c *gin.Context) {
	robotsTxt := `User-agent: *
Allow: /
Disallow: /api/
Disallow: /dashboard/
Disallow: /login
Disallow: /register
Disallow: /forgot-password
Disallow: /reset-password
Disallow: /installer
`
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(robotsTxt))
}

func (s *Server) registerRoutes() {
	apiGroup := s.engine.Group("/api")
	apiGroup.GET("/health", s.healthHandler)
	s.registerInstallerRoutes(apiGroup)
	s.registerAuthRoutes(apiGroup)
	s.registerOAuthRoutes(apiGroup)
	s.registerPasskeyRoutes(apiGroup)
	s.registerAccountRoutes(apiGroup)
	s.registerTicketRoutes(apiGroup)
	s.registerAdminRoutes(apiGroup)
	s.registerShopRoutes(apiGroup)
	s.registerFileRoutes(apiGroup)
	s.registerSiteRoutes(apiGroup)
	s.registerLskyV1Routes(apiGroup)
	s.registerStaticAssets()
	s.registerFrontend()
	s.engine.GET("/robots.txt", s.robotsHandler)
}

func (s *Server) registerFrontend() {
	distPath := filepath.Clean(s.cfg.FrontendDist)
	assetsPath := filepath.Join(distPath, "assets")
	s.engine.StaticFS("/assets", gin.Dir(assetsPath, false))

	s.engine.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
			c.Next()
			return
		}
		c.Next()
	})

	s.engine.NoRoute(middleware.OptionalAuth(s.users, s.session), func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "api route not found",
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
				"message": "check API base path, HTTP method, and trailing slash",
			})
			return
		}
		if s.tryServeLocalFile(c) {
			return
		}
		if s.isKnownFrontendRoute(c.Request.URL.Path) {
			// 安装完成后访问 /installer 返回 404
			if c.Request.URL.Path == "/installer" {
				status, err := s.installer.Status(c.Request.Context())
				if err == nil && status.Installed {
					s.serveIndexHTML(c, distPath, http.StatusNotFound)
					return
				}
			}
			s.serveIndexHTML(c, distPath, http.StatusOK)
			return
		}
		s.serveIndexHTML(c, distPath, http.StatusNotFound)
	})
}

func (s *Server) isKnownFrontendRoute(rawPath string) bool {
	cleanPath := "/" + strings.Trim(strings.TrimSpace(rawPath), "/")
	if cleanPath == "/" {
		return true
	}

	exactRoutes := map[string]struct{}{
		"/installer":       {},
		"/login":           {},
		"/register":        {},
		"/forgot-password": {},
		"/reset-password":  {},
		"/terms":           {},
		"/privacy":         {},
		"/shop":            {},
	}
	if _, ok := exactRoutes[cleanPath]; ok {
		return true
	}

	prefixRoutes := []string{
		"/dashboard",
		"/u",
	}
	for _, prefix := range prefixRoutes {
		if cleanPath == prefix || strings.HasPrefix(cleanPath, prefix+"/") {
			return true
		}
	}

	return false
}

func (s *Server) registerStaticAssets() {
	mounted := make(map[string]struct{})
	reserved := reservedPublicPathSegments()
	registerPath := func(prefix string) {
		prefix = strings.Trim(prefix, "/")
		lowerPrefix := strings.ToLower(prefix)
		if prefix == "" {
			return
		}
		if _, banned := reserved[lowerPrefix]; banned {
			return
		}
		path := "/" + prefix
		if _, exists := mounted[path]; exists {
			return
		}
		s.publicPaths[prefix] = struct{}{}
		handler := func(c *gin.Context) {
			rel := strings.TrimPrefix(c.Param("filepath"), "/")
			if rel == "" || !s.serveLocalFileByRelative(c, rel) {
				c.Status(http.StatusNotFound)
				return
			}
		}
		// 添加可选认证中间件，用于演示站私有图片访问控制
		s.engine.GET(path+"/*filepath", middleware.OptionalAuth(s.users, s.session), handler)
		s.engine.HEAD(path+"/*filepath", middleware.OptionalAuth(s.users, s.session), handler)
		mounted[path] = struct{}{}
	}

	registerPath(s.defaultLocalPublicSegment())

	strategies, err := s.admin.ListStrategies(context.Background())
	if err != nil {
		log.Printf("load strategies for public paths: %v", err)
		return
	}
	for _, strategy := range strategies {
		var cfg map[string]interface{}
		if len(strategy.Configs) == 0 {
			continue
		}
		if err := json.Unmarshal(strategy.Configs, &cfg); err != nil {
			log.Printf("parse strategy config for static mount: %v", err)
			continue
		}
		driver := strings.ToLower(stringValue(cfg, "driver"))
		if driver == "" {
			driver = "local"
		}
		if driver != "local" {
			continue
		}
		baseURL := pathPrefix(stringValue(cfg, "url"))
		if baseURL == "" {
			baseURL = pathPrefix(stringValue(cfg, "base_url"))
		}
		if baseURL == "" {
			baseURL = pathPrefix(stringValue(cfg, "baseUrl"))
		}
		if baseURL == "" {
			baseURL = s.defaultLocalPublicSegment()
		}
		registerPath(baseURL)
	}
}

func pathPrefix(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "//") {
		raw = "http:" + raw
	}
	if strings.HasPrefix(raw, "/") {
		return strings.Trim(strings.Trim(raw, "/"), "/")
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err == nil {
			return strings.Trim(strings.Trim(u.Path, "/"), "/")
		}
	}
	if looksLikeHost(raw) {
		u, err := url.Parse("http://" + raw)
		if err == nil {
			return strings.Trim(strings.Trim(u.Path, "/"), "/")
		}
	}
	return strings.Trim(strings.Trim(raw, "/"), "/")
}

// rejectIfConsoleDomainMismatch returns true when the request host is not the site console host.
// Thumbnails must only be served from the console domain.
func (s *Server) rejectIfConsoleDomainMismatch(c *gin.Context) bool {
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		return false
	}
	consoleURL := strings.TrimSpace(settings["site.console_url"])
	if consoleURL == "" {
		consoleURL = strings.TrimSpace(s.cfg.PublicBaseURL)
	}
	if consoleURL == "" {
		consoleURL = defaultConsoleURL
	}
	expectedHosts := extractConfigHosts(consoleURL)
	if len(expectedHosts) == 0 {
		return false
	}
	actual := requestHostname(c)
	for _, expectedHost := range expectedHosts {
		if hostsMatch(expectedHost, actual) {
			return false
		}
	}
	c.Status(http.StatusNotFound)
	return true
}

// rejectIfStrategyDomainMismatch returns true when the request was rejected (404).
// Only enforces when the strategy explicitly configures an external access domain.
func (s *Server) rejectIfStrategyDomainMismatch(c *gin.Context, strategyID uint) bool {
	if strategyID == 0 {
		return false
	}
	strategy, err := s.admin.FindStrategyByID(c.Request.Context(), strategyID)
	if err != nil {
		return false
	}
	var cfg map[string]interface{}
	if len(strategy.Configs) > 0 {
		if err := json.Unmarshal(strategy.Configs, &cfg); err != nil {
			return false
		}
	}
	domain := strings.TrimSpace(stringValue(cfg, "url"))
	if domain == "" {
		domain = strings.TrimSpace(stringValue(cfg, "base_url"))
	}
	if domain == "" {
		domain = strings.TrimSpace(stringValue(cfg, "baseUrl"))
	}
	expectedHosts := extractConfigHosts(domain)
	if len(expectedHosts) == 0 {
		return false
	}
	actual := requestHostname(c)
	for _, expectedHost := range expectedHosts {
		if hostsMatch(expectedHost, actual) {
			return false
		}
	}
	c.Status(http.StatusNotFound)
	return true
}

func extractConfigHosts(raw string) []string {
	items := splitDomainList(raw)
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		host := extractConfigHost(item)
		if host == "" {
			continue
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		out = append(out, host)
	}
	return out
}

func splitDomainList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = strings.ReplaceAll(raw, "；", ";")
	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func extractConfigHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Relative path is not a custom domain.
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return ""
	}
	normalized := raw
	if strings.HasPrefix(normalized, "//") {
		normalized = "http:" + normalized
	}
	if !strings.Contains(normalized, "://") {
		if !looksLikeHost(normalized) {
			return ""
		}
		normalized = "http://" + normalized
	}
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return normalizeHost(parsed.Host)
}

func requestHostname(c *gin.Context) string {
	host := strings.TrimSpace(c.Request.Host)
	if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-Host")); forwarded != "" {
		// Take the first host if multiple are present.
		if idx := strings.IndexByte(forwarded, ','); idx >= 0 {
			forwarded = strings.TrimSpace(forwarded[:idx])
		}
		if forwarded != "" {
			host = forwarded
		}
	}
	return normalizeHost(host)
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}
	if h, p, err := net.SplitHostPort(host); err == nil {
		if p == "80" || p == "443" {
			return strings.ToLower(h)
		}
		return strings.ToLower(h) + ":" + p
	}
	return host
}

func hostsMatch(expected, actual string) bool {
	if expected == "" || actual == "" {
		return false
	}
	if expected == actual {
		return true
	}
	expHost, expPort, expErr := net.SplitHostPort(expected)
	actHost, actPort, actErr := net.SplitHostPort(actual)
	if expErr != nil {
		expHost = expected
		expPort = ""
	} else {
		expHost = strings.ToLower(expHost)
	}
	if actErr != nil {
		actHost = actual
		actPort = ""
	} else {
		actHost = strings.ToLower(actHost)
	}
	if expHost != actHost {
		return false
	}
	if expPort == "" || actPort == "" {
		return true
	}
	return expPort == actPort
}

func looksLikeHost(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.Contains(raw, ".") || strings.Contains(raw, ":") || strings.HasPrefix(lower, "localhost")
}

func stringValue(cfg map[string]interface{}, key string) string {
	if v, ok := cfg[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (s *Server) defaultLocalPublicSegment() string {
	segment := filepath.Base(s.cfg.StoragePath)
	segment = strings.Trim(segment, "/")
	if segment == "" || segment == "." {
		return "uploads"
	}
	if _, banned := reservedPublicPathSegments()[strings.ToLower(segment)]; banned {
		return "uploads"
	}
	return segment
}

func reservedPublicPathSegments() map[string]struct{} {
	return map[string]struct{}{
		"api":             {},
		"assets":          {},
		"forgot-password": {},
		"reset-password":  {},
		"u":               {},
	}
}

func (s *Server) tryServeLocalFile(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}
	rel := strings.Trim(c.Request.URL.Path, "/")
	if rel == "" {
		return false
	}

	candidates := []string{rel}
	segment, rest := splitFirstSegment(rel)
	if _, ok := s.publicPaths[segment]; ok && rest != "" {
		candidates = append(candidates, rest)
	}

	for _, candidate := range candidates {
		if s.tryServeTicketAttachment(c, candidate) {
			return true
		}
		if s.serveLocalFileByRelative(c, candidate) {
			return true
		}
	}
	return false
}

// tryServeTicketAttachment serves private ticket attachments (console domain + login).
func (s *Server) tryServeTicketAttachment(c *gin.Context, rel string) bool {
	if s.tickets == nil {
		return false
	}
	rel = strings.Trim(strings.TrimPrefix(rel, "/"), "/")
	att, err := s.tickets.FindAttachmentByRelativePath(c.Request.Context(), rel)
	if err != nil {
		return false
	}
	user, ok := middleware.CurrentUser(c)
	if !ok || !s.tickets.CanAccessAttachment(att, &user) {
		c.Status(http.StatusNotFound)
		return true
	}
	if s.rejectIfConsoleDomainMismatch(c) {
		return true
	}
	header := c.Writer.Header()
	// Private resource: do not open CORS to arbitrary origins.
	header.Del("Access-Control-Allow-Origin")
	header.Del("Access-Control-Allow-Credentials")
	header.Set("Cache-Control", "private, no-store")
	header.Set("X-Content-Type-Options", "nosniff")
	mimeType := strings.TrimSpace(att.MimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	header.Set("Content-Type", mimeType)
	header.Set("Content-Disposition", contentDispositionInline(att.Name))

	obj, err := s.files.OpenStoredObject(c.Request.Context(), att.StrategyID, att.Path, att.RelativePath, att.StorageProvider)
	if err != nil {
		// Fall back to local path serve only for local absolute paths.
		if att.Path != "" && filepath.IsAbs(att.Path) {
			c.File(att.Path)
			return true
		}
		c.Status(http.StatusNotFound)
		return true
	}
	defer obj.Body.Close()
	if obj.ContentType != "" {
		header.Set("Content-Type", obj.ContentType)
	}
	if obj.ContentLength > 0 {
		header.Set("Content-Length", strconv.FormatInt(obj.ContentLength, 10))
	}
	c.Status(http.StatusOK)
	if c.Request.Method != http.MethodHead {
		_, _ = io.Copy(c.Writer, obj.Body)
	}
	return true
}

// contentDispositionInline builds a safe Content-Disposition value (no CRLF/quote injection).
func contentDispositionInline(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '\x00' || r == '"' || r == '\\' {
			return -1
		}
		if r < 32 {
			return -1
		}
		return r
	}, name)
	if name == "" {
		name = "attachment"
	}
	if len(name) > 180 {
		name = name[:180]
	}
	// ASCII fallback + RFC 5987 filename* for non-ASCII.
	ascii := strings.Map(func(r rune) rune {
		if r > 127 {
			return '_'
		}
		return r
	}, name)
	if ascii == "" {
		ascii = "attachment"
	}
	return "inline; filename=\"" + ascii + "\"; filename*=UTF-8''" + url.PathEscape(name)
}

func (s *Server) serveLocalFileByRelative(c *gin.Context, rel string) bool {
	// 设置宽松的 CORS 头，允许跨域访问图片/文件资源
	// 这对于图片在第三方网站的嵌入、Canvas 操作和 fetch 请求是必需的
	header := c.Writer.Header()
	header.Set("Access-Control-Allow-Origin", "*")
	header.Del("Access-Control-Allow-Credentials")
	header.Set("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
	header.Set("Access-Control-Allow-Headers", "Content-Type, Range")
	header.Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Content-Disposition, ETag, Last-Modified, Cache-Control")
	header.Set("Cross-Origin-Resource-Policy", "cross-origin")

	file, isThumbnail, err := s.files.FindServeTargetByRelativePath(c.Request.Context(), rel)
	if err != nil {
		return false
	}

	// Thumbnails: login required; only owner or admin may view; console domain only.
	// Fail closed with 404 (no auth/permission hints).
	if isThumbnail {
		user, ok := middleware.CurrentUser(c)
		if !ok || !files.CanAccessThumbnail(file, &user) {
			c.Status(http.StatusNotFound)
			return true
		}
		if s.rejectIfConsoleDomainMismatch(c) {
			return true
		}
	}

	file = files.ServeTarget(file, isThumbnail)

	// Ticket attachments are handled earlier via tryServeTicketAttachment.

	// 原图：策略配置了外部访问域名时，仅允许通过该域名访问，否则 404。
	// 缩略图固定走控制台域名，不跟随策略自定义域名。
	if !isThumbnail {
		if s.rejectIfStrategyDomainMismatch(c, file.StrategyID) {
			return true
		}
	}

	// 演示站模式：私有图片需要登录才能查看
	s.mu.RLock()
	demoMode := s.cfg.DemoMode
	s.mu.RUnlock()

	if demoMode && strings.ToLower(strings.TrimSpace(file.Visibility)) == "private" {
		// 检查用户是否登录
		user, ok := middleware.CurrentUser(c)
		if !ok {
			// 未登录用户无法访问私有图片
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "需要登录才能查看图片",
				"message": "演示站模式下，图片仅对登录用户可见",
			})
			return true // 返回 true 表示已处理请求
		}
		// 检查是否是图片所有者或管理员
		if file.UserID != user.ID && !user.IsAdmin {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "无权访问此私有图片",
				"message": "私有图片仅对上传者和管理员可见",
			})
			return true
		}
	}

	// 移除 visibility 检查 - 公开和私有图片都可以通过直接链接访问
	// visibility 只影响是否在画廊中显示
	driver := strings.ToLower(strings.TrimSpace(file.StorageProvider))
	if driver == "" {
		driver = "local"
	}
	if driver == "s3" || driver == "minio" {
		proxy, err := s.files.FetchProxyObject(c.Request.Context(), file)
		if err != nil {
			return false
		}
		defer proxy.Body.Close()

		if proxy.CacheControl != "" {
			c.Writer.Header().Set("Cache-Control", proxy.CacheControl)
		}
		if proxy.ETag != "" {
			c.Writer.Header().Set("ETag", proxy.ETag)
		}
		if proxy.ContentLength > 0 {
			c.Writer.Header().Set("Content-Length", strconv.FormatInt(proxy.ContentLength, 10))
		}
		if proxy.LastModified != nil {
			c.Writer.Header().Set("Last-Modified", proxy.LastModified.UTC().Format(http.TimeFormat))
		}

		mimeType := strings.TrimSpace(proxy.ContentType)
		if mimeType == "" || mimeType == "application/octet-stream" {
			if strings.TrimSpace(file.MimeType) != "" && file.MimeType != "application/octet-stream" {
				mimeType = file.MimeType
			} else {
				ext := strings.ToLower(strings.TrimPrefix(file.Extension, "."))
				if ext == "" {
					ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name), "."))
				}
				mimeType = getMimeTypeByExtension(ext)
				if mimeType == "" {
					mimeType = "application/octet-stream"
				}
			}
		}
		c.Writer.Header().Set("Content-Type", mimeType)
		if strings.HasPrefix(mimeType, "image/") ||
			strings.HasPrefix(mimeType, "video/") ||
			strings.HasPrefix(mimeType, "audio/") ||
			mimeType == "application/pdf" ||
			strings.HasPrefix(mimeType, "text/") {
			c.Writer.Header().Set("Content-Disposition", "inline")
		}

		c.Status(http.StatusOK)
		if c.Request.Method == http.MethodHead {
			return true
		}
		_, _ = io.Copy(c.Writer, proxy.Body)
		return true
	}
	if driver == "webdav" {
		return s.serveWebDAVFile(c, file)
	}
	if strings.TrimSpace(file.Path) == "" {
		return false
	}
	info, err := os.Stat(file.Path)
	if err != nil || info.IsDir() {
		return false
	}

	// 优先使用数据库中存储的 MimeType，但如果是 application/octet-stream 则重新检测
	mimeType := strings.TrimSpace(file.MimeType)
	if mimeType == "" || mimeType == "application/octet-stream" {
		// 根据文件扩展名检测 MIME 类型
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Path), "."))
		mimeType = getMimeTypeByExtension(ext)
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	// 设置 Content-Type
	c.Writer.Header().Set("Content-Type", mimeType)

	// 对于图片、视频、音频、PDF 等可预览的文件，设置为 inline
	if strings.HasPrefix(mimeType, "image/") ||
		strings.HasPrefix(mimeType, "video/") ||
		strings.HasPrefix(mimeType, "audio/") ||
		mimeType == "application/pdf" ||
		strings.HasPrefix(mimeType, "text/") {
		c.Writer.Header().Set("Content-Disposition", "inline")
	}

	c.File(file.Path)
	return true
}

func (s *Server) serveWebDAVFile(c *gin.Context, file data.FileAsset) bool {
	strategy, err := s.admin.FindStrategyByID(c.Request.Context(), file.StrategyID)
	if err != nil {
		return false
	}
	var cfg map[string]interface{}
	if len(strategy.Configs) > 0 {
		if err := json.Unmarshal(strategy.Configs, &cfg); err != nil {
			return false
		}
	}

	endpoint := strings.TrimSpace(stringValue(cfg, "webdav_endpoint"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(stringValue(cfg, "webdav_url"))
	}
	if endpoint == "" {
		endpoint = strings.TrimSpace(stringValue(cfg, "webdavUrl"))
	}
	username := strings.TrimSpace(stringValue(cfg, "webdav_username"))
	if username == "" {
		username = strings.TrimSpace(stringValue(cfg, "webdav_user"))
	}
	if username == "" {
		username = strings.TrimSpace(stringValue(cfg, "webdavUsername"))
	}
	password := strings.TrimSpace(stringValue(cfg, "webdav_password"))
	if password == "" {
		password = strings.TrimSpace(stringValue(cfg, "webdav_pass"))
	}
	if password == "" {
		password = strings.TrimSpace(stringValue(cfg, "webdavPassword"))
	}
	basePath := strings.TrimSpace(stringValue(cfg, "webdav_base_path"))
	if basePath == "" {
		basePath = strings.TrimSpace(stringValue(cfg, "webdav_path"))
	}
	if basePath == "" {
		basePath = strings.TrimSpace(stringValue(cfg, "webdavBasePath"))
	}
	skipTLSVerify := boolValue(cfg["webdav_skip_tls_verify"]) || boolValue(cfg["webdavSkipTLSVerify"])

	objectURL := strings.TrimSpace(file.Path)
	if !strings.HasPrefix(strings.ToLower(objectURL), "http://") && !strings.HasPrefix(strings.ToLower(objectURL), "https://") {
		remoteURL, err := buildWebDAVObjectURL(endpoint, basePath, file.RelativePath)
		if err != nil {
			return false
		}
		objectURL = remoteURL
	}

	client := &http.Client{}
	if skipTLSVerify {
		tr := http.DefaultTransport.(*http.Transport).Clone()
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		} else {
			tr.TLSClientConfig = tr.TLSClientConfig.Clone()
			tr.TLSClientConfig.InsecureSkipVerify = true
		}
		client = &http.Client{Transport: tr}
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, objectURL, nil)
	if err != nil {
		return false
	}
	if username != "" {
		req.SetBasicAuth(username, password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}

	copyHeaderIfPresent(c.Writer.Header(), resp.Header, "Content-Type")
	copyHeaderIfPresent(c.Writer.Header(), resp.Header, "Content-Length")
	copyHeaderIfPresent(c.Writer.Header(), resp.Header, "Cache-Control")
	copyHeaderIfPresent(c.Writer.Header(), resp.Header, "ETag")
	copyHeaderIfPresent(c.Writer.Header(), resp.Header, "Last-Modified")
	// 不复制 Content-Disposition 头，让浏览器根据 Content-Type 决定是预览还是下载
	// 对于图片等媒体文件，浏览器会自动预览而不是下载

	// 获取或检测正确的 MIME 类型
	mimeType := c.Writer.Header().Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		// 优先使用数据库中的 MimeType
		if strings.TrimSpace(file.MimeType) != "" && file.MimeType != "application/octet-stream" {
			mimeType = file.MimeType
		} else {
			// 根据文件扩展名检测
			ext := strings.ToLower(strings.TrimPrefix(file.Extension, "."))
			if ext == "" {
				ext = strings.ToLower(strings.TrimPrefix(filepath.Ext(file.Name), "."))
			}
			detected := getMimeTypeByExtension(ext)
			if detected != "" {
				mimeType = detected
			} else {
				mimeType = "application/octet-stream"
			}
		}
		c.Writer.Header().Set("Content-Type", mimeType)
	}

	// 确保图片、视频、音频、PDF 等可预览的文件设置为 inline 显示
	if strings.HasPrefix(mimeType, "image/") ||
		strings.HasPrefix(mimeType, "video/") ||
		strings.HasPrefix(mimeType, "audio/") ||
		mimeType == "application/pdf" ||
		strings.HasPrefix(mimeType, "text/") {
		c.Writer.Header().Set("Content-Disposition", "inline")
	}

	c.Status(http.StatusOK)
	if c.Request.Method == http.MethodHead {
		return true
	}
	_, _ = io.Copy(c.Writer, resp.Body)
	return true
}

func buildWebDAVObjectURL(endpoint string, basePath string, relativePath string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", http.ErrMissingFile
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", http.ErrMissingFile
	}
	rel := strings.Trim(strings.ReplaceAll(strings.TrimSpace(relativePath), "\\", "/"), "/")
	parsed.Path = urlPathJoin(parsed.Path, basePath, rel)
	return parsed.String(), nil
}

func urlPathJoin(parts ...string) string {
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		clean := strings.Trim(strings.TrimSpace(part), "/")
		if clean == "" {
			continue
		}
		items = append(items, clean)
	}
	if len(items) == 0 {
		return "/"
	}
	return "/" + strings.Join(items, "/")
}

func boolValue(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		normalized := strings.ToLower(strings.TrimSpace(v))
		return normalized == "1" || normalized == "true" || normalized == "yes" || normalized == "on"
	default:
		return false
	}
}

func copyHeaderIfPresent(dst http.Header, src http.Header, key string) {
	value := strings.TrimSpace(src.Get(key))
	if value == "" {
		return
	}
	dst.Set(key, value)
}

func getMimeTypeByExtension(ext string) string {
	// 常见图片格式
	imageTypes := map[string]string{
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"webp": "image/webp",
		"bmp":  "image/bmp",
		"svg":  "image/svg+xml",
		"ico":  "image/x-icon",
		"tiff": "image/tiff",
		"tif":  "image/tiff",
		"heic": "image/heic",
		"heif": "image/heif",
	}

	// 常见视频格式
	videoTypes := map[string]string{
		"mp4":  "video/mp4",
		"webm": "video/webm",
		"ogg":  "video/ogg",
		"avi":  "video/x-msvideo",
		"mov":  "video/quicktime",
		"wmv":  "video/x-ms-wmv",
		"flv":  "video/x-flv",
		"mkv":  "video/x-matroska",
	}

	// 常见音频格式
	audioTypes := map[string]string{
		"mp3":  "audio/mpeg",
		"wav":  "audio/wav",
		"ogg":  "audio/ogg",
		"m4a":  "audio/mp4",
		"flac": "audio/flac",
		"aac":  "audio/aac",
	}

	// 其他常见格式
	otherTypes := map[string]string{
		"pdf":  "application/pdf",
		"txt":  "text/plain",
		"html": "text/html",
		"htm":  "text/html",
		"css":  "text/css",
		"js":   "application/javascript",
		"json": "application/json",
		"xml":  "application/xml",
		"zip":  "application/zip",
		"rar":  "application/x-rar-compressed",
		"7z":   "application/x-7z-compressed",
	}

	ext = strings.ToLower(strings.TrimSpace(ext))

	if mime, ok := imageTypes[ext]; ok {
		return mime
	}
	if mime, ok := videoTypes[ext]; ok {
		return mime
	}
	if mime, ok := audioTypes[ext]; ok {
		return mime
	}
	if mime, ok := otherTypes[ext]; ok {
		return mime
	}

	return ""
}

func splitFirstSegment(path string) (string, string) {
	path = strings.Trim(path, "/")
	if path == "" {
		return "", ""
	}
	idx := strings.Index(path, "/")
	if idx < 0 {
		return path, ""
	}
	return path[:idx], path[idx+1:]
}

func (s *Server) serveIndexHTML(c *gin.Context, distPath string, statusCode int) {
	indexPath := filepath.Join(distPath, "index.html")

	// 读取 index.html 内容
	content, err := os.ReadFile(indexPath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	// 获取站点配置
	settings, err := s.admin.GetSettings(c.Request.Context())
	if err != nil {
		// 如果获取配置失败，直接返回原始 HTML
		c.Data(statusCode, "text/html; charset=utf-8", content)
		return
	}

	title := strings.TrimSpace(settings["site.title"])
	logo := strings.TrimSpace(settings["site.logo"])

	// 替换 HTML 中的标题
	html := string(content)
	if title != "" {
		safeTitle := stdhtml.EscapeString(title)
		html = strings.Replace(html, "<title>skyImage</title>", "<title>"+safeTitle+"</title>", 1)
	}

	// 替换 favicon
	if logo != "" {
		logoURL := sanitizeFaviconURL(logo)
		if logoURL != "" {
			oldFavicon := `<link rel="icon" type="image/x-icon" href="/favicon.ico" />`
			newFavicon := `<link rel="icon" type="image/x-icon" href="` + stdhtml.EscapeString(logoURL) + `" />`
			html = strings.Replace(html, oldFavicon, newFavicon, 1)
		}
	}

	c.Data(statusCode, "text/html; charset=utf-8", []byte(html))
}

func sanitizeFaviconURL(raw string) string {
	logo := strings.TrimSpace(raw)
	if logo == "" || strings.ContainsAny(logo, "\r\n") {
		return ""
	}
	lower := strings.ToLower(logo)
	if strings.HasPrefix(lower, "data:") {
		if strings.HasPrefix(lower, "data:image/") {
			return logo
		}
		return ""
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		parsed, err := url.Parse(logo)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return ""
		}
		return parsed.String()
	}
	if strings.HasPrefix(logo, "/") {
		return logo
	}
	return "/" + strings.TrimLeft(logo, "/")
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		s.mu.RLock()
		userService := s.users
		sessionManager := s.session
		s.mu.RUnlock()
		middleware.Auth(userService, sessionManager)(c)
	}
}

func (s *Server) optionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		s.mu.RLock()
		userService := s.users
		sessionManager := s.session
		s.mu.RUnlock()
		middleware.OptionalAuth(userService, sessionManager)(c)
	}
}
