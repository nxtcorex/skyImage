package api

import (
	"context"
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
		"site.notfound_mode":    "template",
		"site.notfound_heading": "Gone fishing",
		"site.notfound_text":    "Nothing <here> & nothing there",
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
		"site.notfound_heading": "</p><script>alert(1)</script>",
	}))

	if strings.Contains(body, "<script>") {
		t.Fatalf("heading must be escaped, got %q", body)
	}
}

func TestRenderImageOnlyNoticeUsesCustomHTMLMode(t *testing.T) {
	body := string(renderImageOnlyNotice(map[string]string{
		"site.notfound_mode": "html",
		"site.notfound_html": `<h2 class="oops">Broken <em>link</em></h2>`,
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
		"site.notfound_mode": "template",
		"site.notfound_html": `<h2>should not appear</h2>`,
	}))

	if strings.Contains(body, "should not appear") {
		t.Fatalf("custom html must only render in html mode: %q", body)
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
	if err := server.db.Create(&data.ConfigEntry{Key: "site.notfound_heading", Value: "Synced Heading"}).Error; err != nil {
		t.Fatalf("seed heading setting: %v", err)
	}
	if err := server.db.Create(&data.ConfigEntry{Key: "site.notfound_text", Value: "Synced text"}).Error; err != nil {
		t.Fatalf("seed text setting: %v", err)
	}

	body := string(server.imageOnlyNoticePage(context.Background()))
	if !strings.Contains(body, "Synced Heading") || !strings.Contains(body, "Synced text") {
		t.Fatalf("expected stored settings to be rendered, got %q", body)
	}
}

// The guard-rendered notice must follow the configured 404 content and keep hardening headers.
func TestImageOnlyDomainNoticeUsesConfiguredPage(t *testing.T) {
	server := newImageOnlyTestServer(t, "http://localhost:8080",
		strategyWithConfigs(1, `{"driver":"local","url":"http://127.0.0.1:8080","image_only_domain":true}`),
	)
	if err := server.db.Create(&data.ConfigEntry{Key: "site.notfound_heading", Value: "Custom 404 Headline"}).Error; err != nil {
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
	if body := recorder.Body.String(); !strings.Contains(body, "Custom 404 Headline") {
		t.Fatalf("expected the configured heading, got %q", body)
	}
	if csp := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "sandbox") {
		t.Fatalf("expected a sandboxed CSP for the notice page, got %q", csp)
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected nosniff on the notice page")
	}
}
