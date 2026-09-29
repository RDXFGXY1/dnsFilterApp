package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/api"
	"github.com/RDXFGXY1/dns-filter-app/internal/autostart"
	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/internal/dns"
	"github.com/RDXFGXY1/dns-filter-app/internal/filter"
	"github.com/RDXFGXY1/dns-filter-app/internal/netdns"
	"github.com/RDXFGXY1/dns-filter-app/pkg/logger"
)

type ServiceStatus struct {
	Running, Dashboard, DNSServer, Filtering, DNSRouted, AutoLaunch bool
	StartedAt                                                       time.Time
	LastError                                                       string
}

type ServiceController struct {
	mu               sync.RWMutex
	root, configPath string
	cfg              *config.Config
	db               *database.DB
	filter           *filter.Engine
	dns              *dns.Server
	api              *api.Server
	status           ServiceStatus
}

func NewServiceController() (*ServiceController, error) {
	root, err := findAppRoot()
	if err != nil {
		return nil, err
	}
	return &ServiceController{root: root, configPath: filepath.Join(root, "configs", "config.yaml")}, nil
}

func findAppRoot() (string, error) {
	candidates := []string{}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd, filepath.Join(cwd, ".."), filepath.Join(cwd, "..", ".."))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, dir, filepath.Join(dir, ".."), filepath.Join(dir, "..", ".."))
	}
	for _, candidate := range candidates {
		root, _ := filepath.Abs(candidate)
		if _, err := os.Stat(filepath.Join(root, "configs", "config.yaml")); err == nil {
			if _, err = os.Stat(filepath.Join(root, "web", "templates")); err == nil {
				return root, nil
			}
		}
	}
	return "", fmt.Errorf("application assets were not found beside the executable")
}

func (c *ServiceController) Root() string { return c.root }

// BrowserURL uses the browser-local hostname; internal health requests keep IP routing.
func (c *ServiceController) BrowserURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	port := 8080
	if c.cfg != nil {
		port = c.cfg.Server.APIPort
	}
	return fmt.Sprintf("http://dnsfilter.localhost:%d", port)
}
func (c *ServiceController) DashboardURL() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.cfg != nil {
		return fmt.Sprintf("http://127.0.0.1:%d", c.cfg.Server.APIPort)
	}
	return "http://127.0.0.1:8080"
}

func (c *ServiceController) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.status.Running {
		return nil
	}
	if err := os.Chdir(c.root); err != nil {
		return err
	}
	cfg, err := config.Load(c.configPath)
	if err != nil {
		return c.fail(err)
	}
	if err = preflightPorts(cfg); err != nil {
		return c.fail(err)
	}
	logger.SetLevel(cfg.Logging.Level)
	if err = os.MkdirAll(filepath.Join(c.root, "data", "logs"), 0755); err != nil {
		return c.fail(err)
	}
	setLocalBootstrapResolver()
	db, err := database.New(resolveFrom(c.root, cfg.Database.Path))
	if err != nil {
		return c.fail(err)
	}
	engine, err := filter.New(cfg, db)
	if err != nil {
		db.Close()
		return c.fail(err)
	}
	dnsServer, err := dns.NewServer(cfg, engine, db)
	if err != nil {
		db.Close()
		return c.fail(err)
	}
	apiServer := api.NewServer(cfg, db, engine, dnsServer)
	c.cfg, c.db, c.filter, c.dns, c.api = cfg, db, engine, dnsServer, apiServer
	c.status = ServiceStatus{Running: true, Dashboard: true, DNSServer: true, Filtering: true, AutoLaunch: autostart.IsEnabled(), StartedAt: time.Now()}
	_, _, c.status.DNSRouted, _ = netdns.Status()
	if cfg.Blocklists.AutoUpdateInterval > 0 {
		go engine.StartAutoUpdate(time.Duration(cfg.Blocklists.AutoUpdateInterval) * time.Hour)
	}
	go c.watch("DNS", dnsServer.Start)
	go c.watch("dashboard", apiServer.Start)
	return nil
}

func resolveFrom(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func preflightPorts(cfg *config.Config) error {
	udpAddr := fmt.Sprintf("%s:%d", cfg.Server.DNSHost, cfg.Server.DNSPort)
	udp, err := net.ListenPacket("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("DNS port %s is unavailable (run as Administrator or close the other service): %w", udpAddr, err)
	}
	_ = udp.Close()
	tcpAddr := fmt.Sprintf("%s:%d", cfg.Server.APIHost, cfg.Server.APIPort)
	tcp, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return fmt.Errorf("dashboard port %s is unavailable: %w", tcpAddr, err)
	}
	return tcp.Close()
}
func setLocalBootstrapResolver() {
	net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "udp", "8.8.8.8:53")
	}}
}
func (c *ServiceController) watch(name string, run func() error) {
	if err := run(); err != nil && err != http.ErrServerClosed {
		c.mu.Lock()
		c.status.LastError = fmt.Sprintf("%s server: %v", name, err)
		if name == "dashboard" {
			c.status.Dashboard = false
		} else if name == "DNS" {
			c.status.DNSServer = false
			c.status.Filtering = false
		}
		c.mu.Unlock()
	}
}
func (c *ServiceController) fail(err error) error { c.status.LastError = err.Error(); return err }

func (c *ServiceController) Stop(restoreDNS bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if restoreDNS {
		if err := netdns.ResetDNS(); err != nil {
			c.status.LastError = err.Error()
			return err
		}
		c.status.DNSRouted = false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	var first error
	if c.dns != nil {
		if err := c.dns.Shutdown(ctx); err != nil {
			first = err
		}
	}
	if c.api != nil {
		if err := c.api.Shutdown(ctx); err != nil && first == nil {
			first = err
		}
	}
	if c.db != nil {
		if err := c.db.Close(); err != nil && first == nil {
			first = err
		}
	}
	c.cfg, c.db, c.filter, c.dns, c.api = nil, nil, nil, nil, nil
	c.status.Running, c.status.Dashboard, c.status.DNSServer, c.status.Filtering = false, false, false, false
	if first != nil {
		c.status.LastError = first.Error()
	}
	return first
}

func (c *ServiceController) SetupDNS() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := netdns.SetDNS(); err != nil {
		return c.fail(err)
	}
	c.status.DNSRouted = true
	return nil
}
func (c *ServiceController) RestoreDNS() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := netdns.ResetDNS(); err != nil {
		return c.fail(err)
	}
	c.status.DNSRouted = false
	return nil
}
func (c *ServiceController) SetAutoLaunch(enabled bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	var err error
	if enabled {
		err = autostart.Enable()
	} else {
		err = autostart.Disable()
	}
	if err != nil {
		return c.fail(err)
	}
	c.status.AutoLaunch = enabled
	return nil
}
func (c *ServiceController) Status() ServiceStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	status := c.status
	if c.dns != nil {
		status.Filtering, _ = c.dns.FilteringStatus()
	}
	return status
}
func (c *ServiceController) SetFiltering(enabled bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.dns != nil {
		c.dns.SetFiltering(enabled)
	}
}
