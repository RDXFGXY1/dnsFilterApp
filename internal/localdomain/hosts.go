package localdomain

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

const marker = "# DNS Filter local dashboard"

func rewriteHosts(original []byte, domain string) ([]byte, error) {
	var out bytes.Buffer
	ready := false
	for _, line := range bytes.SplitAfter(original, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		text := strings.TrimSpace(string(line))
		if strings.HasSuffix(text, marker) {
			continue
		}
		active := strings.SplitN(text, "#", 2)[0]
		fields := strings.Fields(active)
		if len(fields) >= 2 {
			for _, host := range fields[1:] {
				if strings.EqualFold(strings.TrimSuffix(host, "."), domain) {
					if fields[0] != "127.0.0.1" {
						return nil, fmt.Errorf("%s already has a conflicting hosts entry", domain)
					}
					ready = true
				}
			}
		}
		out.Write(line)
	}
	if domain != "" && !ready {
		if out.Len() > 0 && !bytes.HasSuffix(out.Bytes(), []byte("\n")) {
			out.WriteString("\r\n")
		}
		out.WriteString("127.0.0.1 " + domain + " " + marker + "\r\n")
	}
	if out.Len() == 0 && domain == "" {
		return nil, errors.New("hosts file is empty")
	}
	return out.Bytes(), nil
}

func containsLocalEntry(content []byte, domain string) bool {
	for _, raw := range bytes.Split(content, []byte("\n")) {
		fields := strings.Fields(strings.SplitN(strings.TrimSpace(string(raw)), "#", 2)[0])
		if len(fields) < 2 || fields[0] != "127.0.0.1" {
			continue
		}
		for _, host := range fields[1:] {
			if strings.EqualFold(strings.TrimSuffix(host, "."), domain) {
				return true
			}
		}
	}
	return false
}
