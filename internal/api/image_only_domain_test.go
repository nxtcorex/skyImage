package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"skyimage/internal/admin"
	"skyimage/internal/config"
	"skyimage/internal/data"
)

func newImageOnlyTestServer(t *testing.T, publicBaseURL string, strategies ...data.Strategy) *Server {
	t.Helper()

	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&data.Strategy{}, &data.ConfigEntry{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	for i := range strategies {
		if err := db.Create(&strategies[i]).Error; err != nil {
			t.Fatalf("failed to insert strategy: %v", err)
		}
	}
	return &Server{
		db:    db,
		admin: admin.New(db),
		cfg:   config.Config{PublicBaseURL: publicBaseURL},
	}
}

func strategyWithConfigs(id uint, configs string) data.Strategy {
	return data.Strategy{
		ID:      id,
		Name:    "strategy",
		Configs: datatypes.JSON([]byte(configs)),
	}
}

func TestCollectImageOnlyDomainHosts(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080",
		strategyWithConfigs(1, `{"driver":"local","url":"https://img.example.com;https://cdn.example.com","image_only_domain":true}`),
		strategyWithConfigs(2, `{"driver":"local","url":"https://free.example.com","image_only_domain":false}`),
		strategyWithConfigs(3, `{"driver":"local","url":"https://legacy.example.com"}`),
	)

	hosts := server.collectSnapshotHosts(t)
	want := map[string]bool{"img.example.com": true, "cdn.example.com": true}
	if len(hosts) != len(want) {
		t.Fatalf("expected %v, got %v", want, hosts)
	}
	for _, host := range hosts {
		if !want[host] {
			t.Fatalf("unexpected host %q in %v", host, hosts)
		}
	}
}

func TestCollectImageOnlyDomainHostsSkipsConsoleHost(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080",
		strategyWithConfigs(1, `{"driver":"local","url":"http://localhost:8080;http://127.0.0.1:8080","image_only_domain":true}`),
	)

	hosts := server.collectSnapshotHosts(t)
	if len(hosts) != 1 || hosts[0] != "127.0.0.1:8080" {
		t.Fatalf("expected only the non-console host to be restricted, got %v", hosts)
	}
}

func TestCollectImageOnlyDomainHostsIgnoresRelativePaths(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080",
		strategyWithConfigs(1, `{"driver":"local","url":"/uploads","image_only_domain":true}`),
	)

	if hosts := server.collectSnapshotHosts(t); len(hosts) != 0 {
		t.Fatalf("expected no restricted hosts for a relative path url, got %v", hosts)
	}
}

// collectSnapshotHosts reads the restriction snapshot without the TTL cache.
func (s *Server) collectSnapshotHosts(t *testing.T) []string {
	t.Helper()
	hosts, err := s.collectImageOnlyDomainHosts(t.Context())
	if err != nil {
		t.Fatalf("collect image-only domains: %v", err)
	}
	return hosts
}

func TestIsFileAccessRequest(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080")

	cases := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/uploads/2026/09/27/a.png", true},
		{http.MethodGet, "/2026/09/27/a.png", true},
		{http.MethodHead, "/2026/09/27/a.png", true},
		{http.MethodGet, "/apicard.png", true},
		{http.MethodGet, "/assets.json", true},
		{http.MethodGet, "/", false},
		{http.MethodGet, "/login", false},
		{http.MethodGet, "/register", false},
		{http.MethodGet, "/installer", false},
		{http.MethodGet, "/dashboard/admin/strategies", false},
		{http.MethodGet, "/u/1", false},
		{http.MethodGet, "/shop", false},
		{http.MethodGet, "/api", false},
		{http.MethodGet, "/assets", false},
		{http.MethodGet, "/api/auth/login", false},
		{http.MethodPost, "/api/auth/login", false},
		{http.MethodPost, "/api/files", false},
		{http.MethodGet, "/api/v1/token", false},
		{http.MethodGet, "/assets/index.js", false},
		{http.MethodGet, "/robots.txt", false},
		{http.MethodGet, "/favicon.ico", false},
		{http.MethodOptions, "/uploads/a.png", false},
	}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(tc.method, tc.path, nil)
		if got := server.isFileAccessRequest(ctx); got != tc.want {
			t.Errorf("%s %s: got %v want %v", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestImageOnlyDomainGuard(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080",
		strategyWithConfigs(1, `{"driver":"local","url":"http://127.0.0.1:8080","image_only_domain":"true"}`),
	)

	engine := gin.New()
	engine.Use(server.imageOnlyDomainGuard)
	engine.GET("/api/auth/login", func(c *gin.Context) { c.String(http.StatusOK, "api") })
	engine.GET("/uploads/*filepath", func(c *gin.Context) { c.String(http.StatusOK, "file") })
	engine.GET("/pic/a.png", func(c *gin.Context) { c.String(http.StatusOK, "file") })
	engine.GET("/login", func(c *gin.Context) { c.String(http.StatusOK, "page") })
	engine.POST("/api/files", func(c *gin.Context) { c.String(http.StatusOK, "upload") })

	cases := []struct {
		host   string
		method string
		path   string
		accept string
		want   int
	}{
		{"127.0.0.1:8080", http.MethodGet, "/pic/a.png", "", http.StatusOK},
		{"127.0.0.1:8080", http.MethodGet, "/uploads/a.png", "", http.StatusOK},
		{"127.0.0.1:8080", http.MethodGet, "/login", "", http.StatusNotFound},
		{"127.0.0.1:8080", http.MethodGet, "/api/auth/login", "", http.StatusNotFound},
		{"127.0.0.1:8080", http.MethodPost, "/api/files", "", http.StatusNotFound},
		{"localhost:8080", http.MethodGet, "/login", "", http.StatusOK},
		{"localhost:8080", http.MethodGet, "/api/auth/login", "", http.StatusOK},
	}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(tc.method, tc.path, nil)
		request.Host = tc.host
		if tc.accept != "" {
			request.Header.Set("Accept", tc.accept)
		}
		engine.ServeHTTP(recorder, request)
		if recorder.Code != tc.want {
			t.Errorf("%s %s on %s: got %d want %d", tc.method, tc.path, tc.host, recorder.Code, tc.want)
		}
	}
}

// A browser navigation on an image-only domain should render a real page instead of the browser's
// own error text, while embedded image requests and API calls keep a body-less 404.
func TestImageOnlyDomainRejectionBody(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080",
		strategyWithConfigs(1, `{"driver":"local","url":"http://127.0.0.1:8080","image_only_domain":true}`),
	)

	engine := gin.New()
	engine.Use(server.imageOnlyDomainGuard)
	engine.GET("/login", func(c *gin.Context) { c.String(http.StatusOK, "page") })
	engine.GET("/pic/a.png", func(c *gin.Context) { c.String(http.StatusOK, "file") })

	navigate := httptest.NewRecorder()
	browserRequest := httptest.NewRequest(http.MethodGet, "/login", nil)
	browserRequest.Host = "127.0.0.1:8080"
	browserRequest.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	engine.ServeHTTP(navigate, browserRequest)

	if navigate.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", navigate.Code)
	}
	if got := navigate.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("expected an html notice, got content type %q", got)
	}
	body := navigate.Body.String()
	if !strings.Contains(body, ">404<") || !strings.Contains(body, "\u6b64\u57df\u540d\u4ec5\u7528\u4e8e\u56fe\u7247\u8bbf\u95ee") {
		t.Fatalf("unexpected notice body: %q", body)
	}
	if strings.Contains(body, "src=") {
		t.Fatalf("notice page must not load external assets: %q", body)
	}

	embedded := httptest.NewRecorder()
	imageRequest := httptest.NewRequest(http.MethodGet, "/login", nil)
	imageRequest.Host = "127.0.0.1:8080"
	imageRequest.Header.Set("Accept", "image/avif,image/webp,*/*")
	engine.ServeHTTP(embedded, imageRequest)
	if embedded.Code != http.StatusNotFound || embedded.Body.Len() != 0 {
		t.Fatalf("expected body-less 404 for non-browser request, got %d with %q", embedded.Code, embedded.Body.String())
	}

	file := httptest.NewRecorder()
	fileRequest := httptest.NewRequest(http.MethodGet, "/pic/a.png", nil)
	fileRequest.Host = "127.0.0.1:8080"
	fileRequest.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	engine.ServeHTTP(file, fileRequest)
	if file.Code != http.StatusOK || file.Body.String() != "file" {
		t.Fatalf("expected stored file requests to pass through, got %d with %q", file.Code, file.Body.String())
	}
}

func TestImageOnlyDomainHostsAreCached(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080",
		strategyWithConfigs(1, `{"driver":"local","url":"http://127.0.0.1:8080","image_only_domain":true}`),
	)

	if !server.isImageOnlyDomain("127.0.0.1:8080") {
		t.Fatal("expected bound domain to be restricted")
	}
	// The snapshot is cached, so removing the strategy has no effect until the cache is invalidated.
	if err := server.db.Delete(&data.Strategy{}, 1).Error; err != nil {
		t.Fatalf("delete strategy: %v", err)
	}
	if !server.isImageOnlyDomain("127.0.0.1:8080") {
		t.Fatal("expected cached snapshot to keep the domain restricted")
	}
	server.invalidateImageOnlyDomainCache()
	if server.isImageOnlyDomain("127.0.0.1:8080") {
		t.Fatal("expected invalidation to drop the restriction")
	}
}

// A fresh install seeds site.console_url with a placeholder, so the runtime public base URL
// must also be exempt or the switch would lock the operator out of the console.
func TestCollectImageOnlyDomainHostsExemptsPublicBaseURL(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://127.0.0.1:8080",
		strategyWithConfigs(1, `{"driver":"local","url":"http://localhost:8080;http://127.0.0.1:8080;http://img.example.com","image_only_domain":true}`),
	)
	if err := server.db.Create(&data.ConfigEntry{Key: "site.console_url", Value: "http://localhost:8080"}).Error; err != nil {
		t.Fatalf("seed console url setting: %v", err)
	}

	hosts := server.collectSnapshotHosts(t)
	if len(hosts) != 1 || hosts[0] != "img.example.com" {
		t.Fatalf("expected only img.example.com to be restricted, got %v", hosts)
	}
}
