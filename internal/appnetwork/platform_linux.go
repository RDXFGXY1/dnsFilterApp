//go:build linux

package appnetwork

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const linuxAppTable = "dnsfilter_apps"

var (
	linuxCgroupRoot = "/sys/fs/cgroup"
	linuxStateFile  = "/run/dns-filter/appnetwork-state.json"
	linuxNftBinary  = "nft"
)

type linuxScope struct {
	Rule     string            `json:"rule"`
	Path     string            `json:"path"`
	Cgroup   string            `json:"cgroup"`
	Origins  map[string]string `json:"origins"`
	Attached map[string]bool   `json:"attached"`
}

type linuxPlatform struct {
	mu     sync.Mutex
	scopes map[string]*linuxScope
}

func newPlatform() platform {
	p := &linuxPlatform{scopes: map[string]*linuxScope{}}
	p.loadState()
	go p.watch()
	return p
}

func (p *linuxPlatform) Supported() bool {
	if os.Geteuid() != 0 {
		return false
	}
	if _, err := exec.LookPath(linuxNftBinary); err != nil {
		return false
	}
	_, err := os.Stat(filepath.Join(linuxCgroupRoot, "cgroup.controllers"))
	return err == nil
}

func validatePathPlatform(path string) error {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Mode()&0111 == 0 {
		return errors.New("select a valid Linux executable file")
	}
	clean := filepath.Clean(path)
	critical := map[string]bool{
		"/usr/lib/systemd/systemd": true,
		"/usr/bin/systemd":         true,
		"/usr/bin/init":            true,
		"/sbin/init":               true,
	}
	if clean == "/proc/1/exe" || critical[clean] {
		return errors.New("this critical Linux process cannot be blocked")
	}
	return nil
}

func (p *linuxPlatform) RunningApps() ([]App, error) {
	return linuxRunningApps()
}

func linuxRunningApps() ([]App, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read Linux processes: %w", err)
	}
	byPath := map[string]App{}
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid < 2 {
			continue
		}
		path, err := os.Readlink(filepath.Join("/proc", entry.Name(), "exe"))
		if err != nil || path == "" {
			continue
		}
		path = strings.TrimSuffix(path, " (deleted)")
		if _, exists := byPath[path]; exists {
			continue
		}
		byPath[path] = App{Name: filepath.Base(path), Path: path, PID: pid, Running: true}
	}
	apps := make([]App, 0, len(byPath))
	for _, app := range byPath {
		apps = append(apps, app)
	}
	sort.Slice(apps, func(i, j int) bool { return strings.ToLower(apps[i].Name) < strings.ToLower(apps[j].Name) })
	return apps, nil
}

func (p *linuxPlatform) Block(path, rule string) error {
	if !p.Supported() {
		return errors.New("Linux App Network Control requires root, cgroup v2, and nftables")
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	scope := p.scopes[rule]
	if scope == nil {
		sum := sha256.Sum256([]byte(rule))
		name := hex.EncodeToString(sum[:8])
		scope = &linuxScope{
			Rule: rule, Path: filepath.Clean(path), Cgroup: "dnsfilter-apps/" + name,
			Origins: map[string]string{}, Attached: map[string]bool{},
		}
		p.scopes[rule] = scope
	}
	if err := os.MkdirAll(filepath.Join(linuxCgroupRoot, scope.Cgroup), 0755); err != nil {
		delete(p.scopes, rule)
		return fmt.Errorf("create application network cgroup: %w", err)
	}
	if err := p.attachMatching(scope); err != nil {
		return err
	}
	if err := p.rebuildRules(); err != nil {
		return err
	}
	return p.saveState()
}

func (p *linuxPlatform) Allow(rule string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	scope := p.scopes[rule]
	if scope == nil {
		// Rebuilding also removes stale nftables rules from a previous process.
		return p.rebuildRules()
	}
	if err := p.restoreScope(scope); err != nil {
		return err
	}
	delete(p.scopes, rule)
	if err := p.rebuildRules(); err != nil {
		p.scopes[rule] = scope
		return err
	}
	_ = os.Remove(filepath.Join(linuxCgroupRoot, scope.Cgroup))
	return p.saveState()
}

func (p *linuxPlatform) watch() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		p.mu.Lock()
		changed := false
		for _, scope := range p.scopes {
			before := len(scope.Attached)
			if p.attachMatching(scope) == nil && len(scope.Attached) != before {
				changed = true
			}
		}
		if changed {
			_ = p.saveState()
		}
		p.mu.Unlock()
	}
}

func (p *linuxPlatform) attachMatching(scope *linuxScope) error {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return err
	}
	self := os.Getpid()
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid < 2 || pid == self {
			continue
		}
		exe, err := os.Readlink(filepath.Join("/proc", entry.Name(), "exe"))
		if err != nil || filepath.Clean(strings.TrimSuffix(exe, " (deleted)")) != scope.Path {
			continue
		}
		pidText := strconv.Itoa(pid)
		current, err := processCgroup(pid)
		if err != nil || strings.TrimPrefix(current, "/") == scope.Cgroup {
			continue
		}
		if _, exists := scope.Origins[pidText]; !exists {
			scope.Origins[pidText] = current
			if err := p.saveState(); err != nil {
				return fmt.Errorf("save application cgroup recovery state: %w", err)
			}
		}
		if err := os.WriteFile(filepath.Join(linuxCgroupRoot, scope.Cgroup, "cgroup.procs"), []byte(pidText), 0600); err != nil {
			return fmt.Errorf("move %s (PID %d) into its network-control scope: %w", filepath.Base(scope.Path), pid, err)
		}
		scope.Attached[pidText] = true
	}
	return nil
}

func processCgroup(pid int) (string, error) {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "cgroup"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "0::") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "0::"))
			if path == "" || !strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
				return "", errors.New("invalid unified cgroup path")
			}
			return path, nil
		}
	}
	return "", errors.New("cgroup v2 process membership is unavailable")
}

func (p *linuxPlatform) restoreScope(scope *linuxScope) error {
	fallback := ""
	for _, origin := range scope.Origins {
		fallback = origin
		break
	}
	raw, err := os.ReadFile(filepath.Join(linuxCgroupRoot, scope.Cgroup, "cgroup.procs"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, pidText := range strings.Fields(string(raw)) {
		origin := scope.Origins[pidText]
		if origin == "" {
			origin = fallback
		}
		if origin == "" || !strings.HasPrefix(origin, "/") || strings.Contains(origin, "..") {
			return fmt.Errorf("cannot safely restore PID %s to its original cgroup", pidText)
		}
		target := filepath.Join(linuxCgroupRoot, strings.TrimPrefix(origin, "/"), "cgroup.procs")
		if _, err := os.Stat(target); os.IsNotExist(err) {
			target = filepath.Join(linuxCgroupRoot, "cgroup.procs")
		}
		if err := os.WriteFile(target, []byte(pidText), 0600); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("restore PID %s from its network-control cgroup: %w", pidText, err)
		}
	}
	return nil
}

func (p *linuxPlatform) rebuildRules() error {
	_ = runLinuxNft("delete", "table", "inet", linuxAppTable)
	if len(p.scopes) == 0 {
		return nil
	}
	var script strings.Builder
	script.WriteString("add table inet dnsfilter_apps\n")
	script.WriteString("add chain inet dnsfilter_apps output { type filter hook output priority -5; policy accept; }\n")
	keys := make([]string, 0, len(p.scopes))
	for key := range p.scopes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		scope := p.scopes[key]
		level := len(strings.Split(scope.Cgroup, "/"))
		fmt.Fprintf(&script, "add rule inet dnsfilter_apps output socket cgroupv2 level %d \"%s\" counter drop comment \"%s\"\n", level, scope.Cgroup, scope.Rule)
	}
	cmd := exec.Command(linuxNftBinary, "-f", "-")
	cmd.Stdin = strings.NewReader(script.String())
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("nftables app control: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func runLinuxNft(args ...string) error {
	out, err := exec.Command(linuxNftBinary, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("nftables app control: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (p *linuxPlatform) loadState() {
	raw, err := os.ReadFile(linuxStateFile)
	if err != nil {
		return
	}
	var scopes map[string]*linuxScope
	if json.Unmarshal(raw, &scopes) != nil {
		return
	}
	for rule, scope := range scopes {
		if scope == nil || scope.Rule != rule || !strings.HasPrefix(scope.Cgroup, "dnsfilter-apps/") || strings.Contains(scope.Cgroup, "..") {
			continue
		}
		if scope.Origins == nil {
			scope.Origins = map[string]string{}
		}
		if scope.Attached == nil {
			scope.Attached = map[string]bool{}
		}
		p.scopes[rule] = scope
	}
}

func (p *linuxPlatform) saveState() error {
	if err := os.MkdirAll(filepath.Dir(linuxStateFile), 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(p.scopes, "", "  ")
	if err != nil {
		return err
	}
	tmp := linuxStateFile + ".tmp"
	if err := os.WriteFile(tmp, raw, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, linuxStateFile)
}
