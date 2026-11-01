package vault

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"
)

func TestRecoveryEmailHasReadableHTMLAndPlainText(t *testing.T) {
	for _, sample := range []struct{ subject, plain, expected string }{
		{"DNS Filter Vault recovery code", "Your one-time Vault recovery code is: 12345678\r\nIt expires in 10 minutes.", "12345678"},
		{"DNS Filter email recovery enabled", "Email recovery is now enabled for this DNS Filter installation.", "Email recovery is ready"},
	} {
		data, err := buildRecoveryMessage("sender@example.com", "owner@example.com", sample.subject, sample.plain)
		if err != nil {
			t.Fatal(err)
		}
		msg, err := mail.ReadMessage(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		kind, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
		if err != nil || kind != "multipart/alternative" {
			t.Fatalf("not multipart: %s %v", kind, err)
		}
		parts := multipart.NewReader(msg.Body, params["boundary"])
		seen := map[string]string{}
		for {
			part, err := parts.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := io.ReadAll(quotedprintable.NewReader(part))
			if err != nil {
				t.Fatal(err)
			}
			mediaType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
			if err != nil {
				t.Fatal(err)
			}
			seen[mediaType] = string(decoded)
		}
		if seen["text/plain"] != sample.plain {
			t.Fatalf("plain fallback changed: %q", seen["text/plain"])
		}
		if !strings.Contains(seen["text/html"], sample.expected) || !strings.Contains(seen["text/html"], "DNS Filter") {
			t.Fatalf("HTML missing key content: %s", seen["text/html"])
		}
	}
}
