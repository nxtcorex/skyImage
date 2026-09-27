package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"skyimage/internal/admin"
	"skyimage/internal/config"
	"skyimage/internal/data"
)

// newInstalledServer boots the real router (console SPA + API + file serving) against a throwaway
// database, with one local strategy bound to img.test.
func newInstalledServer(t *testing.T, imageOnly bool) *Server {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "skyimage.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := data.AutoMigrateAll(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("database handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	storageDir := t.TempDir()
	distDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(distDir, "index.html"), []byte("<html>console</html>"), 0o600); err != nil {
		t.Fatalf("write index: %v", err)
	}

	cfg := config.Config{
		HTTPAddr:      ":0",
		PublicBaseURL: "http://console.test",
		StoragePath:   filepath.ToSlash(storageDir),
		FrontendDist:  distDir,
	}
	server := NewServer(cfg, db)

	imageOnlyValue := interface{}(false)
	if imageOnly {
		imageOnlyValue = true
	}
	if _, err := server.admin.CreateStrategy(t.Context(), admin.StrategyPayload{
		Key:  10,
		Name: "image-cdn",
		Configs: map[string]interface{}{
			"driver":            "local",
			"root":              filepath.ToSlash(storageDir),
			"url":               "http://img.test",
			"image_only_domain": imageOnlyValue,
		},
	}); err != nil {
		t.Fatalf("create strategy: %v", err)
	}

	if err := os.MkdirAll(filepath.Join(storageDir, "2026", "09"), 0o755); err != nil {
		t.Fatalf("prepare storage dir: %v", err)
	}
	body := []byte("fake-png-bytes")
	if err := os.WriteFile(filepath.Join(storageDir, "2026", "09", "pic.png"), body, 0o600); err != nil {
		t.Fatalf("write image: %v", err)
	}
	var strategy data.Strategy
	if err := db.First(&strategy).Error; err != nil {
		t.Fatalf("load strategy: %v", err)
	}
	file := data.FileAsset{
		UserID:          1,
		StrategyID:      strategy.ID,
		Key:             "k1",
		Path:            filepath.Join(storageDir, "2026", "09", "pic.png"),
		RelativePath:    "2026/09/pic.png",
		Name:            "pic.png",
		OriginalName:    "pic.png",
		Size:            int64(len(body)),
		MimeType:        "image/png",
		Extension:       "png",
		Visibility:      "public",
		StorageProvider: "local",
	}
	if err := db.Create(&file).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	server.invalidateImageOnlyDomainCache()
	return server
}

func doRequest(t *testing.T, server *Server, method, host, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader(body)
	}
	request := httptest.NewRequest(method, path, reader)
	request.Host = host
	recorder := httptest.NewRecorder()
	server.engine.ServeHTTP(recorder, request)
	return recorder
}

func TestImageOnlyDomainEndToEnd(t *testing.T) {
	server := newInstalledServer(t, true)

	cases := []struct {
		name   string
		method string
		host   string
		path   string
		body   []byte
		want   int
	}{
		{"image served", http.MethodGet, "img.test", "/2026/09/pic.png", nil, http.StatusOK},
		{"image head", http.MethodHead, "img.test", "/2026/09/pic.png", nil, http.StatusOK},
		{"missing image still 404", http.MethodGet, "img.test", "/2026/09/none.png", nil, http.StatusNotFound},
		{"login page blocked", http.MethodGet, "img.test", "/login", nil, http.StatusNotFound},
		{"register page blocked", http.MethodGet, "img.test", "/register", nil, http.StatusNotFound},
		{"home page blocked", http.MethodGet, "img.test", "/", nil, http.StatusNotFound},
		{"dashboard blocked", http.MethodGet, "img.test", "/dashboard/admin/strategies", nil, http.StatusNotFound},
		{"app assets blocked", http.MethodGet, "img.test", "/assets/app.js", nil, http.StatusNotFound},
		{"login api blocked", http.MethodPost, "img.test", "/api/auth/login", []byte(`{}`), http.StatusNotFound},
		{"register api blocked", http.MethodPost, "img.test", "/api/auth/register", []byte(`{}`), http.StatusNotFound},
		{"site config api blocked", http.MethodGet, "img.test", "/api/site/config", nil, http.StatusNotFound},
		{"upload api blocked", http.MethodPost, "img.test", "/api/files", []byte(`{}`), http.StatusNotFound},
		{"console keeps pages", http.MethodGet, "console.test", "/login", nil, http.StatusOK},
		{"console keeps api", http.MethodGet, "console.test", "/api/site/config", nil, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := doRequest(t, server, tc.method, tc.host, tc.path, tc.body)
			if recorder.Code != tc.want {
				t.Fatalf("%s %s://%s: got %d want %d (%s)", tc.method, tc.host, tc.path, recorder.Code, tc.want, recorder.Body.String())
			}
		})
	}

	if got := doRequest(t, server, http.MethodGet, "img.test", "/2026/09/pic.png", nil).Body.String(); got != "fake-png-bytes" {
		t.Fatalf("unexpected image body: %q", got)
	}
	if body := doRequest(t, server, http.MethodGet, "img.test", "/login", nil).Body.String(); strings.Contains(body, "console") {
		t.Fatalf("console html leaked on image-only domain: %q", body)
	}
}

func TestImageOnlyDomainSwitchOffKeepsDomainUsable(t *testing.T) {
	server := newInstalledServer(t, false)

	if code := doRequest(t, server, http.MethodGet, "img.test", "/login", nil).Code; code != http.StatusOK {
		t.Fatalf("expected console page when switch is off, got %d", code)
	}
	if code := doRequest(t, server, http.MethodGet, "img.test", "/api/site/config", nil).Code; code != http.StatusOK {
		t.Fatalf("expected api when switch is off, got %d", code)
	}
}
