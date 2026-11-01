package vault

import (
	"bytes"
	"fmt"
	"html"
	"mime/multipart"
	"mime/quotedprintable"
	"net/textproto"
	"strings"
)

func buildRecoveryMessage(from, to, subject, plain string) ([]byte, error) {
	var message bytes.Buffer
	if _, err := fmt.Fprintf(&message, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\n", from, to, subject); err != nil {
		return nil, err
	}
	parts := multipart.NewWriter(&message)
	if _, err := fmt.Fprintf(&message, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", parts.Boundary()); err != nil {
		return nil, err
	}
	for _, part := range []struct{ contentType, body string }{
		{"text/plain; charset=UTF-8", plain},
		{"text/html; charset=UTF-8", recoveryHTML(subject, plain)},
	} {
		header := textproto.MIMEHeader{}
		header.Set("Content-Type", part.contentType)
		header.Set("Content-Transfer-Encoding", "quoted-printable")
		writer, err := parts.CreatePart(header)
		if err != nil {
			return nil, err
		}
		encoded := quotedprintable.NewWriter(writer)
		if _, err = encoded.Write([]byte(part.body)); err != nil {
			return nil, err
		}
		if err = encoded.Close(); err != nil {
			return nil, err
		}
	}
	if err := parts.Close(); err != nil {
		return nil, err
	}
	return message.Bytes(), nil
}

func recoveryHTML(subject, plain string) string {
	codeMail := subject == "DNS Filter Vault recovery code"
	title := "Email recovery is ready"
	intro := "Your DNS Filter Vault can now send one-time recovery codes to this address. Keep your offline recovery code somewhere safe as a second way back in."
	badge := "RECOVERY ENABLED"
	callout := `<tr><td style="padding:0 36px 28px"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#eff6ff;border:1px solid #c7d8ff;border-radius:12px"><tr><td style="padding:18px 20px;color:#183b78;font-size:14px;line-height:22px">When you need to reset your Vault password, open <strong>Settings → System &amp; Security → Configuration Vault</strong> and request a new code.</td></tr></table></td></tr>`
	if codeMail {
		title = "Reset your Vault password"
		intro = "Use this one-time code in DNS Filter to reset your Configuration Vault password."
		badge = "VAULT RECOVERY"
		code := strings.TrimSpace(strings.TrimPrefix(strings.SplitN(plain, "\r\n", 2)[0], "Your one-time Vault recovery code is: "))
		callout = `<tr><td style="padding:0 36px 24px"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#edf2ff;border:1px solid #c9d5ff;border-radius:12px"><tr><td align="center" style="padding:22px 12px 6px;color:#53627e;font-size:12px;font-weight:700;letter-spacing:2px">YOUR ONE-TIME CODE</td></tr><tr><td align="center" style="padding:0 12px 22px;color:#1d3557;font-family:Consolas,Menlo,monospace;font-size:34px;font-weight:800;letter-spacing:7px">` + html.EscapeString(code) + `</td></tr></table></td></tr>`
	}
	footer := "This message confirms the recovery email you chose."
	if codeMail {
		footer = "This code expires in 10 minutes. If you did not request it, you can ignore this email."
	}
	return `<!doctype html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` + html.EscapeString(title) + `</title></head>` +
		`<body style="margin:0;padding:0;background:#f3f5f9;color:#172033;font-family:Arial,Helvetica,sans-serif">` +
		`<div style="display:none;font-size:1px;line-height:1px;color:#f3f5f9;max-height:0;max-width:0;opacity:0;overflow:hidden">` + html.EscapeString(intro) + `</div>` +
		`<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#f3f5f9"><tr><td align="center" style="padding:32px 12px">` +
		`<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="width:100%;max-width:600px;background:#ffffff;border:1px solid #e0e5ee;border-radius:16px;overflow:hidden">` +
		`<tr><td style="background:#101827;padding:24px 36px;color:#ffffff"><span style="display:inline-block;width:30px;height:30px;line-height:30px;text-align:center;background:#4d6ee8;border-radius:8px;font-size:18px;font-weight:800">D</span><span style="padding-left:10px;font-size:17px;font-weight:800;vertical-align:middle">DNS Filter</span></td></tr>` +
		`<tr><td style="padding:34px 36px 12px"><span style="color:#4b63c9;font-size:11px;font-weight:800;letter-spacing:2px">` + badge + `</span><h1 style="margin:12px 0 0;color:#172033;font-size:28px;line-height:35px">` + html.EscapeString(title) + `</h1></td></tr>` +
		`<tr><td style="padding:0 36px 26px;color:#536079;font-size:15px;line-height:24px">` + html.EscapeString(intro) + `</td></tr>` + callout +
		`<tr><td style="padding:0 36px 34px;color:#6b7487;font-size:13px;line-height:21px">` + html.EscapeString(footer) + `</td></tr>` +
		`<tr><td style="border-top:1px solid #e8ebf2;padding:20px 36px;color:#8992a3;font-size:11px;line-height:18px">Sent by your local DNS Filter installation. No reply is needed.</td></tr>` +
		`</table></td></tr></table></body></html>`
}
