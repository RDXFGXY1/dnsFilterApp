//go:build windows

package connectionfreeze

import (
	"fmt"
	"github.com/tailscale/wf"
	"golang.org/x/sys/windows"
	"os"
	"time"
	"unsafe"
)

func supported() bool { return true }

type windowsBackend struct {
	s       *wf.Session
	filters map[uint64]bool
	seen    map[string]time.Time
	since   time.Time
}

func startPlatform() (backend, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return startWindows(exe)
}
func startWindows(exempt string) (_ backend, err error) {
	s, err := wf.New(&wf.Options{Name: "DNS Filter Connection Freeze", Description: "Temporary block of new outbound connections", Dynamic: true})
	if err != nil {
		return nil, fmt.Errorf("Windows connection control requires Administrator permission: %w", err)
	}
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	guid, err := windows.GenerateGUID()
	if err != nil {
		return nil, err
	}
	sub := wf.SublayerID(guid)
	if err = s.AddSublayer(&wf.Sublayer{ID: sub, Name: "DNS Filter Connection Freeze", Weight: 0xffff}); err != nil {
		return nil, err
	}
	app, err := wf.AppID(exempt)
	if err != nil {
		return nil, err
	}
	ids := map[wf.RuleID]bool{}
	for _, layer := range []wf.LayerID{wf.LayerALEAuthConnectV4, wf.LayerALEAuthConnectV6} {
		guid, err := windows.GenerateGUID()
		if err != nil {
			return nil, err
		}
		rule := &wf.Rule{ID: wf.RuleID(guid), Name: "Freeze new outbound connections", Layer: layer, Sublayer: sub, Weight: 1, Action: wf.ActionBlock,
			Conditions: []*wf.Match{
				{Field: wf.FieldFlags, Op: wf.MatchTypeFlagsNoneSet, Value: uint32(wf.ConditionFlagIsLoopback | wf.ConditionFlagIsReauthorize)},
				{Field: wf.FieldALEAppID, Op: wf.MatchTypeNotEqual, Value: app},
			}}
		if err = s.AddRule(rule); err != nil {
			return nil, err
		}
		ids[rule.ID] = true
		// Keep DHCP renewals possible without granting other system traffic access.
		local, remote := uint16(68), uint16(67)
		if layer == wf.LayerALEAuthConnectV6 {
			local, remote = 546, 547
		}
		guid, err = windows.GenerateGUID()
		if err != nil {
			return nil, err
		}
		if err = s.AddRule(&wf.Rule{ID: wf.RuleID(guid), Name: "Keep DHCP renewals", Layer: layer, Sublayer: sub, Weight: 10, Action: wf.ActionPermit,
			Conditions: []*wf.Match{{Field: wf.FieldIPProtocol, Op: wf.MatchTypeEqual, Value: uint8(17)}, {Field: wf.FieldIPLocalPort, Op: wf.MatchTypeEqual, Value: local}, {Field: wf.FieldIPRemotePort, Op: wf.MatchTypeEqual, Value: remote}}}); err != nil {
			return nil, err
		}
	}
	filters := map[uint64]bool{}
	for id := range ids {
		kernelID, e := kernelRuleID(id)
		if e != nil {
			return nil, e
		}
		filters[kernelID] = true
	}
	if len(filters) != 2 {
		return nil, fmt.Errorf("could not verify both IPv4 and IPv6 freeze rules")
	}
	return &windowsBackend{s: s, filters: filters, seen: map[string]time.Time{}, since: time.Now()}, nil
}

// The upstream wrapper models FWPM_FILTER0's context union as consecutive fields.
// Read the native structure with the correct union size to obtain real event IDs.
type nativeValue struct {
	Type  uint32
	Value uintptr
}
type nativeBlob struct {
	Size uint32
	Data uintptr
}
type nativeFilter struct {
	Key               windows.GUID
	Name, Description uintptr
	Flags             uint32
	Provider          uintptr
	Data              nativeBlob
	Layer, Sublayer   windows.GUID
	Weight            nativeValue
	Count             uint32
	Conditions        uintptr
	Action            uint32
	ActionKey         windows.GUID
	Context           [2]uint64
	Reserved          uintptr
	ID                uint64
	EffectiveWeight   nativeValue
}

func kernelRuleID(id wf.RuleID) (uint64, error) {
	dll := windows.NewLazySystemDLL("fwpuclnt.dll")
	var handle windows.Handle
	code, _, _ := dll.NewProc("FwpmEngineOpen0").Call(0, 10, 0, 0, uintptr(unsafe.Pointer(&handle)))
	if code != 0 {
		return 0, fmt.Errorf("open WFP verifier: 0x%x", code)
	}
	defer dll.NewProc("FwpmEngineClose0").Call(uintptr(handle))
	var filter *nativeFilter
	code, _, _ = dll.NewProc("FwpmFilterGetByKey0").Call(uintptr(handle), uintptr(unsafe.Pointer(&id)), uintptr(unsafe.Pointer(&filter)))
	if code != 0 {
		return 0, fmt.Errorf("verify WFP rule: 0x%x", code)
	}
	defer dll.NewProc("FwpmFreeMemory0").Call(uintptr(unsafe.Pointer(&filter)))
	return filter.ID, nil
}
func (b *windowsBackend) Close() error { return b.s.Close() }
func (b *windowsBackend) Events() ([]Event, error) {
	cutoff := time.Now().Add(-5 * time.Minute)
	if cutoff.After(b.since) {
		b.since = cutoff
	}
	for key, stamp := range b.seen {
		if stamp.Before(b.since) {
			delete(b.seen, key)
		}
	}
	drops, err := b.s.DropEvents()
	if err != nil {
		return nil, err
	}
	events := []Event{}
	for _, d := range drops {
		if !b.filters[d.FilterID] || d.Timestamp.Before(b.since) {
			continue
		}
		key := fmt.Sprintf("%d/%d/%s/%s/%s", d.Timestamp.UnixNano(), d.FilterID, d.AppID, d.LocalAddr, d.RemoteAddr)
		if _, ok := b.seen[key]; ok {
			continue
		}
		b.seen[key] = d.Timestamp
		protocol := fmt.Sprint(d.IPProtocol)
		switch d.IPProtocol {
		case 6:
			protocol = "TCP"
		case 17:
			protocol = "UDP"
		case 1:
			protocol = "ICMP"
		case 58:
			protocol = "ICMPv6"
		}
		events = append(events, Event{Time: d.Timestamp.UTC().Format(time.RFC3339Nano), Action: "blocked", App: d.AppID, Local: d.LocalAddr.String(), Remote: d.RemoteAddr.String(), Protocol: protocol, Reason: "New outbound connection while Connection Freeze is active"})
	}
	// Deduplication is bounded to the last five minutes of Windows events.
	return events, nil
}
