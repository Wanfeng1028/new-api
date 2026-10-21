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
const brandedEmailTagline = "One key is all you need"

// Tagline variant used in the welcome-mail subject line.
const brandedEmailSubjectTagline = "One key is all you need"

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
		`<div style="font-family:'Times New Roman',Times,serif,'PingFang SC','Microsoft YaHei',Arial,sans-serif;font-size:24px;font-weight:700;color:#181410;letter-spacing:0.5px;">` + name + `</div>` +
		`<div style="font-family:'Times New Roman',Times,serif,'PingFang SC','Microsoft YaHei',Arial,sans-serif;font-size:12px;color:#8A8175;letter-spacing:2px;">` + brandedEmailTagline + `</div>` +
		`</td></tr></table></td></tr>`)
	b.WriteString(`<tr><td style="padding:22px 36px 0 36px;"><div style="height:3px;background-color:#D9481C;font-size:0;line-height:0;">&nbsp;</div></td></tr>`)
	b.WriteString(`<tr><td style="padding:26px 36px 28px 36px;font-family:'Times New Roman',Times,serif,'PingFang SC','Microsoft YaHei',Arial,sans-serif;">` +
		`<div style="font-size:20px;font-weight:700;color:#181410;">` + html.EscapeString(heading) + `</div>` + body +
		`</td></tr>`)
	b.WriteString(`<tr><td style="padding:0 36px;"><div style="border-top:1px solid #E7DFD2;padding:16px 0 22px 0;font-family:'Times New Roman',Times,serif,'PingFang SC','Microsoft YaHei',Arial,sans-serif;font-size:12px;color:#B4AB9C;text-align:center;">` +
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
// paragraphs (password reset, security notices); plain links get the
// brand color. Anchors that already carry a style attribute (such as
// the capsule buttons) are left untouched.
func RenderNoticeEmail(heading string, bodyHTML string) string {
	bodyHTML = strings.ReplaceAll(bodyHTML, "<a href=", `<a style="color:#D9481C;" href=`)
	body := `<div style="margin-top:6px;font-size:15px;line-height:1.7;color:#3D362C;">` + bodyHTML + `</div>`
	return brandedEmailShell(heading, body)
}

// RenderWelcomeEmail renders the registration welcome mail: a greeting, a
// three-step quick start, the API base URL in a card, and two capsule
// buttons linking to the console and the integration docs. It returns the
// subject and the full HTML body.
func RenderWelcomeEmail(username string) (string, string) {
	subject := "欢迎加入 " + SystemName + "，" + brandedEmailSubjectTagline
	step := func(num string, title string, desc string) string {
		return `<tr><td style="padding:7px 0;"><table role="presentation" cellpadding="0" cellspacing="0"><tr>` +
			`<td style="vertical-align:top;"><span style="display:inline-block;width:26px;height:26px;line-height:26px;border-radius:50%;background-color:#D9481C;color:#FFFFFF;font-size:14px;font-weight:700;text-align:center;">` + num + `</span></td>` +
			`<td style="vertical-align:top;padding-left:12px;">` +
			`<div style="font-size:15px;font-weight:700;color:#181410;">` + title + `</div>` +
			`<div style="font-size:13.5px;line-height:1.7;color:#3D362C;">` + desc + `</div>` +
			`</td></tr></table></td></tr>`
	}
	adminMail := `<a href="mailto:wanfeng572@gmail.com" style="color:#D9481C;">wanfeng572@gmail.com</a>`
	var b strings.Builder
	b.WriteString(`<p style="margin:14px 0 0 0;font-size:15px;line-height:1.7;color:#3D362C;">你好，<b>` + html.EscapeString(username) + `</b>！你的账号已经创建成功，欢迎来到 ` + html.EscapeString(SystemName) + `——` + brandedEmailTagline + `</p>`)
	b.WriteString(`<div style="margin-top:20px;font-size:15px;font-weight:700;color:#181410;">三步快速开始</div>`)
	b.WriteString(`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="margin-top:6px;">`)
	b.WriteString(step("1", "联系管理员开通额度", `新账号默认额度为 0，请联系管理员开通后即可使用（管理员邮箱：`+adminMail+`）。`))
	b.WriteString(step("2", "在控制台创建你的 API 令牌", `进入控制台，在「令牌」页面创建你的 API 令牌，调用接口时用它作为密钥。`))
	b.WriteString(step("3", "改一个地址即可接入", `把应用里的 API 地址（Base URL）改为下方地址，再填入令牌，即可开始调用：`))
	b.WriteString(`</table>`)
	b.WriteString(`<div style="margin-top:10px;background-color:#FAF7F1;border:2px solid #181410;padding:14px 10px;text-align:center;">` +
		`<span style="font-family:'SF Mono',Consolas,Menlo,monospace;font-size:16px;font-weight:700;color:#181410;">https://api.botsarena.tech/v1</span></div>`)
	b.WriteString(`<div style="margin-top:22px;text-align:center;">` +
		`<a href="https://dashboard.botsarena.tech/console" style="display:inline-block;background-color:#D9481C;color:#FFFFFF;font-size:15px;font-weight:700;text-decoration:none;padding:12px 30px;border-radius:999px;border:2px solid #D9481C;">进入控制台</a>` +
		`<span style="display:inline-block;width:12px;font-size:0;">&nbsp;</span>` +
		`<a href="https://botsrouter.botsarena.tech/#/docs" style="display:inline-block;background-color:#FFFFFF;color:#181410;font-size:15px;font-weight:700;text-decoration:none;padding:12px 30px;border-radius:999px;border:2px solid #181410;">接入文档</a>` +
		`</div>`)
	b.WriteString(`<p style="margin:20px 0 0 0;font-size:13px;line-height:1.7;color:#8A8175;">使用中遇到任何问题，都可以联系管理员邮箱 ` + adminMail + `。</p>`)
	return subject, brandedEmailShell("欢迎加入 "+SystemName, b.String())
}
