package api

// 图片专用域名的 404 提示页。
//
// 控制台的可配置 404 页是 SPA 路由，需要 index.html、/assets/* 与 /api/site/config，
// 而这三者正是「域名仅限图片访问」开关要拦掉的内容，所以这里用一张自包含的内联页面，
// 只复用后台配置的文案：模板模式取「404 大字 + 描述」，自定义 HTML 模式取管理员编写的 HTML。
// 后台 HTML 在前端由 DOMPurify 过滤，服务端这里使用同一份白名单，避免多出一个注入点。

import (
	"context"
	"html/template"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// 与控制台 NotFoundPage 中 DOMPurify 的 ALLOWED_TAGS / ALLOWED_ATTR 保持一致。
var noticeAllowedTags = map[string]struct{}{
	"p": {}, "div": {}, "section": {}, "h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {}, "h6": {},
	"ul": {}, "ol": {}, "li": {}, "strong": {}, "em": {}, "br": {}, "a": {}, "span": {},
}

// 整体丢弃子树的标签：白名单里没有它们，保留文本还可能带出脚本或样式。
var noticeDiscardedTags = map[string]struct{}{
	"script": {}, "style": {}, "noscript": {}, "template": {}, "iframe": {}, "frame": {},
	"frameset": {}, "object": {}, "embed": {}, "link": {}, "meta": {}, "base": {}, "form": {},
	"input": {}, "button": {}, "select": {}, "option": {}, "textarea": {}, "svg": {}, "math": {},
}

const (
	noticeFallbackHeading = "404"
	noticeFallbackTitle   = "此域名仅用于图片访问"
	noticeFallbackDesc    = "This domain only serves stored image and file requests.\n请通过站点主域名访问控制台。"
)

const noticePageStyle = `body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;background:#f8fafc;color:#0f172a;font-family:system-ui,-apple-system,"Segoe UI",Roboto,"PingFang SC","Microsoft YaHei",sans-serif}` +
	`main{max-width:640px;padding:32px;text-align:center}` +
	`.heading{margin:0;font-size:60px;line-height:1;font-weight:700;letter-spacing:-.02em}` +
	`.title{margin:16px 0 8px;font-size:18px;font-weight:600}` +
	`.desc{margin:0;font-size:14px;line-height:1.7;color:#475569;white-space:pre-wrap}` +
	`.desc a{color:#2563eb}`

// imageOnlyNoticePage 按后台的 404 设置渲染提示页；读取失败或留空时退回默认提示。
func (s *Server) imageOnlyNoticePage(ctx context.Context) []byte {
	settings, err := s.admin.GetSettings(ctx)
	if err != nil {
		settings = nil
	}
	return renderImageOnlyNotice(settings)
}

func renderImageOnlyNotice(settings map[string]string) []byte {
	title := noticePageTitle(settings)
	if htmlBody := sanitizeNoticeHTML(settings["site.notfound_html"]); settings["site.notfound_mode"] == "html" && htmlBody != "" {
		return buildNoticePage(title, htmlBody)
	}

	heading := noticeFallbackHeading
	if value := strings.TrimSpace(settings["site.notfound_heading"]); value != "" {
		heading = template.HTMLEscapeString(value)
	}

	var body strings.Builder
	body.WriteString(`<p class="heading">` + heading + `</p>`)
	if value := strings.TrimSpace(settings["site.notfound_text"]); value != "" {
		body.WriteString(`<div class="desc">` + template.HTMLEscapeString(value) + `</div>`)
		return buildNoticePage(title, body.String())
	}
	body.WriteString(`<h2 class="title">` + noticeFallbackTitle + `</h2>`)
	body.WriteString(`<p class="desc">` + template.HTMLEscapeString(noticeFallbackDesc) + `</p>`)
	return buildNoticePage(title, body.String())
}

// noticePageTitle 与控制台一致：404 页把文档标题换成站点名称。
func noticePageTitle(settings map[string]string) string {
	if value := strings.TrimSpace(settings["site.title"]); value != "" {
		return template.HTMLEscapeString(value)
	}
	return "404 · 无法访问"
}

func buildNoticePage(pageTitle, body string) []byte {
	if pageTitle == "" {
		pageTitle = "404"
	}
	return []byte(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<title>` + pageTitle + `</title>
<style>` + noticePageStyle + `</style>
</head>
<body>
<main>
` + body + `
</main>
</body>
</html>`)
}

// sanitizeNoticeHTML 解析后台 HTML 并只重写出白名单内的节点，未成对的标签由解析器补全，
// 因此输出中不存在脚本、事件属性与危险协议。
func sanitizeNoticeHTML(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	contextNode := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(raw), contextNode)
	if err != nil {
		return ""
	}
	var sb strings.Builder
	for _, node := range nodes {
		writeNoticeNode(&sb, node)
	}
	return sb.String()
}

func writeNoticeNode(sb *strings.Builder, node *html.Node) {
	switch node.Type {
	case html.TextNode:
		sb.WriteString(template.HTMLEscapeString(node.Data))
	case html.ElementNode:
		writeNoticeElement(sb, node)
	case html.CommentNode:
		// 注释可能包裹条件包含与脚本片段，直接丢弃。
	}
}

func writeNoticeElement(sb *strings.Builder, node *html.Node) {
	tag := strings.ToLower(node.Data)
	if _, discarded := noticeDiscardedTags[tag]; discarded {
		return
	}
	_, allowed := noticeAllowedTags[tag]
	if allowed {
		attrs := noticeAttributes(tag, node.Attr)
		sb.WriteString("<" + tag)
		for _, attr := range attrs {
			sb.WriteString(" " + attr.Key + `="` + template.HTMLEscapeString(attr.Val) + `"`)
		}
		sb.WriteString(">")
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		writeNoticeNode(sb, child)
	}
	if allowed && tag != "br" {
		sb.WriteString("</" + tag + ">")
	}
}

// noticeAttributes 只保留白名单属性；新开窗口一律补 noopener，忽略后台填写的 rel。
func noticeAttributes(tag string, raw []html.Attribute) []html.Attribute {
	var out []html.Attribute
	targetBlank := false
	for _, attr := range raw {
		value, ok := noticeAttributeValue(tag, strings.ToLower(attr.Key), attr.Val)
		if !ok {
			continue
		}
		attr.Key = strings.ToLower(attr.Key)
		attr.Val = value
		if attr.Key == "target" {
			targetBlank = true
			continue
		}
		if attr.Key == "rel" {
			continue
		}
		out = append(out, attr)
	}
	if targetBlank {
		out = append(out, html.Attribute{Key: "target", Val: "_blank"}, html.Attribute{Key: "rel", Val: "noopener noreferrer"})
	}
	return out
}

// noticeAttributeValue 校验单个属性的取值。
func noticeAttributeValue(tag, name, rawValue string) (string, bool) {
	switch name {
	case "class":
		return strings.TrimSpace(rawValue), true
	case "href":
		if tag != "a" {
			return "", false
		}
		value := strings.TrimSpace(stripControlChars(rawValue))
		parsed, err := url.Parse(value)
		if err != nil {
			return "", false
		}
		switch strings.ToLower(parsed.Scheme) {
		case "", "http", "https", "mailto":
			return value, true
		}
		return "", false
	case "target":
		if tag == "a" && strings.EqualFold(strings.TrimSpace(rawValue), "_blank") {
			return "_blank", true
		}
		return "", false
	}
	return "", false
}

// stripControlChars 移除会被浏览器在解析协议前忽略的控制字符，防止 java\tscript: 之类的绕过。
func stripControlChars(value string) string {
	return strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' || r == '\f' || r == 0 {
			return -1
		}
		return r
	}, value)
}
