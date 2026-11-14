package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"skyimage/internal/data"
)

func TestRenderImageOnlyNoticeFallsBackToTheImageOnlyHint(t *testing.T) {
	body := string(renderImageOnlyNotice(map[string]string{"site.title": "Sky"}))

	if !strings.Contains(body, `<title>Sky</title>`) {
		t.Fatalf("expected the site title in the document title, got %q", body)
	}
	if !strings.Contains(body, noticeFallbackHeading) || !strings.Contains(body, noticeFallbackTitle) {
		t.Fatalf("expected the fallback notice, got %q", body)
	}
	if strings.Contains(body, "<script") || strings.Contains(body, "src=") {
		t.Fatalf("notice must stay self-contained: %q", body)
	}
}

func TestRenderImageOnlyNoticeUsesConfiguredText(t *testing.T) {
	body := string(renderImageOnlyNotice(map[string]string{
		"site.imageonly_mode":    "template",
		"site.imageonly_heading": "Gone fishing",
		"site.imageonly_text":    "Nothing <here> & nothing there",
	}))

	if !strings.Contains(body, "Gone fishing") {
		t.Fatalf("expected the configured heading, got %q", body)
	}
	if !strings.Contains(body, "Nothing &lt;here&gt; &amp; nothing there") {
		t.Fatalf("expected escaped configured text, got %q", body)
	}
	if strings.Contains(body, noticeFallbackTitle) {
		t.Fatalf("configured text should replace the fallback hint: %q", body)
	}
}

func TestRenderImageOnlyNoticeEscapesHeading(t *testing.T) {
	body := string(renderImageOnlyNotice(map[string]string{
		"site.imageonly_heading": "</p><script>alert(1)</script>",
	}))

	if strings.Contains(body, "<script>") {
		t.Fatalf("heading must be escaped, got %q", body)
	}
}

func TestRenderImageOnlyNoticeUsesCustomHTMLMode(t *testing.T) {
	body := string(renderImageOnlyNotice(map[string]string{
		"site.imageonly_mode": "html",
		"site.imageonly_html": `<h2 class="oops">Broken <em>link</em></h2>`,
	}))

	if !strings.Contains(body, `<h2 class="oops">Broken <em>link</em></h2>`) {
		t.Fatalf("expected the sanitized custom html, got %q", body)
	}
	if strings.Contains(body, noticeFallbackTitle) {
		t.Fatalf("custom html should replace the fallback hint: %q", body)
	}
}

func TestRenderImageOnlyNoticeIgnoresCustomHTMLOutsideHtmlMode(t *testing.T) {
	body := string(renderImageOnlyNotice(map[string]string{
		"site.imageonly_mode": "template",
		"site.imageonly_html": `<h2>should not appear</h2>`,
	}))

	if strings.Contains(body, "should not appear") {
		t.Fatalf("custom html must only render in html mode: %q", body)
	}
}

func TestRenderImageOnlyNoticeKeepsNotfoundSettingsUnused(t *testing.T) {
	body := string(renderImageOnlyNotice(map[string]string{
		"site.notfound_mode":    "html",
		"site.notfound_html":    `<h1>console 404</h1>`,
		"site.notfound_heading": "console heading",
		"site.notfound_text":    "console text",
	}))

	if strings.Contains(body, "console 404") || strings.Contains(body, "console heading") || strings.Contains(body, "console text") {
		t.Fatalf("image-only notice must use its own settings, not the console 404 ones: %q", body)
	}
	if !strings.Contains(body, noticeFallbackTitle) {
		t.Fatalf("expected the default image-only fallback notice, got %q", body)
	}
}

func TestRenderImageOnlyNoticeSupportsDarkMode(t *testing.T) {
	body := string(renderImageOnlyNotice(map[string]string{}))

	if !strings.Contains(body, "prefers-color-scheme: dark") {
		t.Fatalf("expected the default template to follow the browser dark theme, got %q", body)
	}
	if !strings.Contains(body, `<meta name="color-scheme" content="light dark">`) {
		t.Fatalf("expected a color-scheme meta tag, got %q", body)
	}
}

func TestSanitizeNoticeHTML(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		contains []string
		excludes []string
	}{
		{
			name:     "keeps whitelisted tags and attributes",
			input:    `<p class="a">Hello <strong>world</strong><br><a href="https://example.com/x?a=1&amp;b=2" target="_blank">link</a></p>`,
			contains: []string{`<p class="a">`, "<strong>world</strong>", "<br>", `href="https://example.com/x?a=1&amp;b=2"`, `rel="noopener noreferrer"`},
			excludes: []string{},
		},
		{
			name:     "drops scripts and their bodies",
			input:    `<div>ok</div><script>alert(1)</script>`,
			contains: []string{"<div>ok</div>"},
			excludes: []string{"alert(1)", "<script"},
		},
		{
			name:     "drops event handlers",
			input:    `<p onclick="steal()" onmouseover="steal()">text</p>`,
			contains: []string{"<p>text</p>"},
			excludes: []string{"onclick", "steal"},
		},
		{
			name:     "drops javascript and data urls",
			input:    `<a href="javascript:alert(1)">a</a><a href="DATA:text/html;base64,PHNjcmlwdD4=">b</a>`,
			contains: []string{">a</a>", ">b</a>"},
			excludes: []string{"javascript:", "PHNjcmlwdD4", "href="},
		},
		{
			name:     "strips control characters before scheme check",
			input:    "<a href=\"java\tscript:alert(1)\">c</a>",
			contains: []string{">c</a>"},
			excludes: []string{"javascript", "alert"},
		},
		{
			name:     "keeps relative and anchor links",
			input:    `<a href="/docs">d</a><a href="#top">t</a>`,
			contains: []string{`href="/docs"`, `href="#top"`},
			excludes: []string{},
		},
		{
			name:     "unwraps disallowed containers but keeps text",
			input:    `<marquee>rolling</marquee>`,
			contains: []string{"rolling"},
			excludes: []string{"<marquee"},
		},
		{
			name:     "drops iframes images and comments",
			input:    `<div>a</div><iframe src="//evil"></iframe><img src=x onerror=alert(1)><!-- secret -->`,
			contains: []string{"<div>a</div>"},
			excludes: []string{"iframe", "//evil", "<img", "secret"},
		},
		{
			name:     "drops style subtrees",
			input:    `<style>body{display:none}</style><p>visible</p>`,
			contains: []string{"<p>visible</p>"},
			excludes: []string{"display:none", "<style"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := sanitizeNoticeHTML(tc.input)
			for _, want := range tc.contains {
				if !strings.Contains(out, want) {
					t.Errorf("expected %q in %q", want, out)
				}
			}
			for _, ban := range tc.excludes {
				if strings.Contains(strings.ToLower(out), strings.ToLower(ban)) {
					t.Errorf("unexpected %q in %q", ban, out)
				}
			}
		})
	}
}

func TestSanitizeNoticeHTMLAlwaysNeutralizesBlankTargets(t *testing.T) {
	out := sanitizeNoticeHTML(`<a href="https://example.com" target="_blank" rel="opener">x</a>`)
	if strings.Contains(out, `rel="opener"`) {
		t.Fatalf("expected the admin rel to be replaced, got %q", out)
	}
	if !strings.Contains(out, `rel="noopener noreferrer"`) {
		t.Fatalf("expected noopener rel, got %q", out)
	}
}

func TestImageOnlyNoticePageReadsStoredSettings(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080")
	if err := server.db.Create(&data.ConfigEntry{Key: "site.imageonly_heading", Value: "Synced Heading"}).Error; err != nil {
		t.Fatalf("seed heading setting: %v", err)
	}
	if err := server.db.Create(&data.ConfigEntry{Key: "site.imageonly_text", Value: "Synced text"}).Error; err != nil {
		t.Fatalf("seed text setting: %v", err)
	}

	body := string(server.imageOnlyNoticePage(context.Background()))
	if !strings.Contains(body, "Synced Heading") || !strings.Contains(body, "Synced text") {
		t.Fatalf("expected stored settings to be rendered, got %q", body)
	}
}

// The guard-rendered notice must follow the configured image-only content and keep hardening headers.
func TestImageOnlyDomainNoticeUsesConfiguredPage(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080",
		strategyWithConfigs(1, `{"driver":"local","url":"http://127.0.0.1:8080","image_only_domain":true}`),
	)
	if err := server.db.Create(&data.ConfigEntry{Key: "site.imageonly_heading", Value: "Custom Notice Headline"}).Error; err != nil {
		t.Fatalf("seed heading setting: %v", err)
	}

	engine := gin.New()
	engine.Use(server.imageOnlyDomainGuard)
	engine.GET("/login", func(c *gin.Context) { c.String(http.StatusOK, "page") })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/login", nil)
	request.Host = "127.0.0.1:8080"
	request.Header.Set("Accept", "text/html,application/xhtml+xml,*/*")
	engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "Custom Notice Headline") {
		t.Fatalf("expected the configured heading, got %q", body)
	}
	if csp := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") {
		t.Fatalf("expected a sandboxed CSP for the notice page, got %q", csp)
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected nosniff on the notice page")
	}
}

// adminUserContextKey 复刻 middleware 的上下文 key（未导出常量），供直接调用处理器时注入超级管理员。
const adminUserContextKey = "currentUser"

func superAdminSettingsContext(method, target, body string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	if body == "" {
		c.Request = httptest.NewRequest(method, target, nil)
	} else {
		c.Request = httptest.NewRequest(method, target, strings.NewReader(body))
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Set(adminUserContextKey, data.User{IsSuperAdmin: true})
	return c, recorder
}

// 管理端保存拦截页设置后，拦截页渲染应读取同一份配置（而非控制台 404 页配置）。
func TestImageOnlyNoticeSettingsRoundTrip(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080")

	c, recorder := superAdminSettingsContext(http.MethodPatch, "/admin/system/site",
		`{"imageOnlyMode":"html","imageOnlyHtml":"<h1>Notice only</h1><script>alert(1)</script>","imageOnlyHeading":"Ignored heading"}`)
	server.handleAdminUpdateSiteSettings(c)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 on update, got %d: %s", recorder.Code, recorder.Body.String())
	}

	body := string(server.imageOnlyNoticePage(context.Background()))
	if !strings.Contains(body, "Notice only") {
		t.Fatalf("expected the saved custom html to render, got %q", body)
	}
	if strings.Contains(body, "<script") || strings.Contains(body, "alert(1)") {
		t.Fatalf("scripts must be filtered out: %q", body)
	}
	if strings.Contains(body, "Ignored heading") {
		t.Fatalf("template heading must not render in html mode: %q", body)
	}

	gc, grecorder := superAdminSettingsContext(http.MethodGet, "/admin/system/site", "")
	server.handleAdminSiteSettings(gc)
	if grecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 on read, got %d: %s", grecorder.Code, grecorder.Body.String())
	}
	var payload struct {
		Data struct {
			ImageOnlyMode    string `json:"imageOnlyMode"`
			ImageOnlyHeading string `json:"imageOnlyHeading"`
			ImageOnlyHtml    string `json:"imageOnlyHtml"`
		} `json:"data"`
	}
	if err := json.Unmarshal(grecorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode settings response: %v", err)
	}
	if payload.Data.ImageOnlyMode != "html" || !strings.Contains(payload.Data.ImageOnlyHtml, "Notice only") || payload.Data.ImageOnlyHeading != "Ignored heading" {
		t.Fatalf("expected saved settings to be returned, got %+v", payload.Data)
	}

	// 非法模式归一化为 template，切回后自定义 HTML 不再渲染
	c2, recorder2 := superAdminSettingsContext(http.MethodPatch, "/admin/system/site", `{"imageOnlyMode":"bogus"}`)
	server.handleAdminUpdateSiteSettings(c2)
	if recorder2.Code != http.StatusOK {
		t.Fatalf("expected 200 on mode update, got %d: %s", recorder2.Code, recorder2.Body.String())
	}
	if body := string(server.imageOnlyNoticePage(context.Background())); strings.Contains(body, "Notice only") {
		t.Fatalf("custom html must not render in template mode: %q", body)
	}
}
