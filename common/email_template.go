package common

import (
	"html"
	"strings"
)

// BotsRouter fork patch: brand shell for system mail. Mail clients cannot
// render SVG, so the header uses the PNG logo served from the dashboard
// brand directory.
const brandedEmailLogoURL = "https://dashboard.botsarena.tech/brand/logo-email.png"
const brandedEmailSiteURL = "https://botsrouter.botsarena.tech"
const brandedEmailTagline = "你的无限 TOKEN 工厂！"

func brandedEmailShell(heading string, body string) string {
	name := html.EscapeString(SystemName)
	heading = strings.TrimSpace(strings.TrimPrefix(heading, SystemName))
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="zh-CN"><head><meta charset="UTF-8">` +
		`<meta name="viewport" content="width=device-width, initial-scale=1.0"></head>` +
		`<body style="margin:0;padding:0;background-color:#FAF7F1;">` +
		`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#FAF7F1;"><tr><td align="center" style="padding:32px 16px;">` +
		`<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;background-color:#FFFFFF;border:2px solid #181410;box-shadow:8px 8px 0 #181410;">`)
	b.WriteString(`<tr><td style="padding:26px 36px 0 36px;"><table role="presentation" cellpadding="0" cellspacing="0"><tr>` +
		`<td style="vertical-align:middle;"><img src="` + brandedEmailLogoURL + `" alt="` + name + `" width="76" height="57" style="display:block;"></td>` +
		`<td style="vertical-align:middle;padding-left:16px;">` +
		`<div style="font-family:'PingFang SC','Microsoft YaHei',Arial,sans-serif;font-size:24px;font-weight:700;color:#181410;letter-spacing:0.5px;">` + name + `</div>` +
		`<div style="font-family:'PingFang SC','Microsoft YaHei',Arial,sans-serif;font-size:12px;color:#8A8175;letter-spacing:2px;">` + brandedEmailTagline + `</div>` +
		`</td></tr></table></td></tr>`)
	b.WriteString(`<tr><td style="padding:22px 36px 0 36px;"><div style="height:3px;background-color:#D9481C;font-size:0;line-height:0;">&nbsp;</div></td></tr>`)
	b.WriteString(`<tr><td style="padding:26px 36px 28px 36px;font-family:'PingFang SC','Microsoft YaHei',Arial,sans-serif;">` +
		`<div style="font-size:20px;font-weight:700;color:#181410;">` + html.EscapeString(heading) + `</div>` + body +
		`</td></tr>`)
	b.WriteString(`<tr><td style="padding:0 36px;"><div style="border-top:1px solid #E7DFD2;padding:16px 0 22px 0;font-family:'PingFang SC','Microsoft YaHei',Arial,sans-serif;font-size:12px;color:#B4AB9C;text-align:center;">` +
		name + ` · <a href="` + brandedEmailSiteURL + `" style="color:#B4AB9C;text-decoration:underline;">botsrouter.botsarena.tech</a><br>` +
		`<span style="color:#C9C0B0;">这封邮件由系统自动发送，请勿直接回复</span></div></td></tr>`)
	b.WriteString(`</table></td></tr></table></body></html>`)
	return b.String()
}

// RenderCodeEmail renders a verification-code mail: an intro line, the code
// in a large card, and a small-print note.
func RenderCodeEmail(heading string, intro string, code string, note string) string {
	body := `<p style="margin:14px 0 0 0;font-size:15px;line-height:1.7;color:#3D362C;">` + html.EscapeString(intro) + `</p>` +
		`<div style="margin-top:18px;background-color:#FAF7F1;border:2px solid #181410;padding:20px 10px;text-align:center;">` +
		`<span style="font-family:'SF Mono',Consolas,Menlo,monospace;font-size:36px;font-weight:700;letter-spacing:10px;color:#181410;">` + html.EscapeString(code) + `</span></div>` +
		`<div style="text-align:center;font-size:12px;color:#8A8175;padding-top:8px;letter-spacing:3px;">验 证 码</div>` +
		`<p style="margin:14px 0 0 0;font-size:13px;line-height:1.7;color:#8A8175;">` + html.EscapeString(note) + `</p>`
	return brandedEmailShell(heading, body)
}

// RenderNoticeEmail renders a mail whose body is ready-made HTML
// paragraphs (password reset, security notices); links get the brand color.
func RenderNoticeEmail(heading string, bodyHTML string) string {
	bodyHTML = strings.ReplaceAll(bodyHTML, "<a ", `<a style="color:#D9481C;" `)
	body := `<div style="margin-top:6px;font-size:15px;line-height:1.7;color:#3D362C;">` + bodyHTML + `</div>`
	return brandedEmailShell(heading, body)
}
