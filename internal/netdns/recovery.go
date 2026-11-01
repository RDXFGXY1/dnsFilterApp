package netdns

import (
	"fmt"
	dnsproto "github.com/miekg/dns"
	"net"
	"strings"
	"time"
)

// RecoverIfOffline releases exclusively loopback DNS only when the local resolver
// stops answering. It does not stop protection, alter firewall rules, or change
// the saved routing preference. DNS Route Watch restores it when the server returns.
func RecoverIfOffline() (bool, error) {
	return recoverIfOffline(Status, localResolverAnswers, ResetDNS)
}
func localResolverAnswers(address string) bool {
	query := new(dnsproto.Msg)
	query.SetQuestion(".", dnsproto.TypeNS)
	// Any valid reply proves a resolver is alive. SERVFAIL/refused are not reasons
	// to bypass a running filter's policies or a temporarily unavailable upstream.
	reply, _, e := (&dnsproto.Client{Timeout: 2 * time.Second}).Exchange(query, net.JoinHostPort(address, "53"))
	return e == nil && reply != nil
}
func recoverIfOffline(status func() (string, string, bool, error), answers func(string) bool, reset func() error) (bool, error) {
	_, servers, local, e := status()
	if e != nil {
		return false, e
	}
	if !local {
		return false, nil
	}
	addresses := strings.FieldsFunc(servers, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' })
	offline := false
	for _, address := range addresses {
		if !answers(address) && !answers(address) {
			offline = true
			break
		}
	}
	if !offline {
		return false, nil
	}
	// Respect an adapter/DNS change made while the probes were running.
	_, current, stillLocal, e := status()
	if e != nil {
		return false, e
	}
	if !stillLocal || current != servers {
		return false, nil
	}
	if e = reset(); e != nil {
		return false, fmt.Errorf("local DNS is unavailable and automatic DNS recovery failed: %w", e)
	}
	return true, nil
}
