package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/api"
	"github.com/RDXFGXY1/dns-filter-app/internal/appnetwork"
	"github.com/RDXFGXY1/dns-filter-app/internal/autostart"
	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/internal/dns"
	"github.com/RDXFGXY1/dns-filter-app/internal/filter"
	"github.com/RDXFGXY1/dns-filter-app/internal/netdns"
	appversion "github.com/RDXFGXY1/dns-filter-app/internal/version"
	"github.com/RDXFGXY1/dns-filter-app/pkg/logger"
)

var (
	configPath = flag.String("config", "./configs/config.yaml", "Path to configuration file")
	devMode    = flag.Bool("dev", false, "Run in development mode")
)

func main() {
	// Make relative asset paths (configs/, web/, data/) resolve no matter how
	// the app is launched — double-click, scheduled task, or from another CWD.
	// Development runs intentionally keep the repository as the working
	// directory so templates, static assets, configuration, and data are read
	// directly from the source tree instead of build/dev.
	if !hasArgument("-dev") {
		chdirToExeDir()
	}

	// Handle subcommands before flag parsing.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "setup-dns":
			runDNSCommand("setup")
			return
		case "reset-dns":
			runDNSCommand("reset")
			return
		case "reset-app-network":
			runAppNetworkReset()
			return
		case "install-autostart":
			if err := autostart.Enable(); err != nil {
				fmt.Printf("Failed to install Windows background startup: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("DNS Filter background startup installed.")
			return
		case "uninstall-autostart":
			if err := autostart.Disable(); err != nil {
				fmt.Printf("Failed to remove Windows background startup: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("DNS Filter background startup removed.")
			return
		case "version", "--version", "-version":
			fmt.Println(appversion.Current)
			return
		}
	}

	flag.Parse()

	printBanner()

	// Initialize logger
	log := logger.New(*devMode)
	log.Infof("Starting DNS Filter Application v%s", appversion.Current)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	logger.SetLevel(cfg.Logging.Level)
	if cfg.Logging.File != "" {
		logger.SetOutput(cfg.Logging.File)
	}

	// Ensure data directory exists
	if err := os.MkdirAll("./data/logs", 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// FIX: Set bootstrap DNS BEFORE initializing filter engine
	// This prevents the chicken-and-egg DNS problem where DNS Filter
	// can't download blocklists because it uses itself for DNS lookups
	setBootstrapDNS()

	// Initialize database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Info("Database initialized successfully")

	// Initialize filter engine (downloads blocklists using bootstrap DNS)
	filterEngine, err := filter.New(cfg, db)
	if err != nil {
		log.Fatalf("Failed to initialize filter engine: %v", err)
	}

	log.Infof("Filter engine initialized with %d blocklist entries", filterEngine.GetBlockedCount())

	// Start blocklist auto-updater
	if cfg.Blocklists.AutoUpdateInterval > 0 {
		go filterEngine.StartAutoUpdate(time.Duration(cfg.Blocklists.AutoUpdateInterval) * time.Hour)
		log.Infof("Blocklist auto-update enabled (every %d hours)", cfg.Blocklists.AutoUpdateInterval)
	}

	// Initialize and start DNS server
	dnsServer, err := dns.NewServer(cfg, filterEngine, db)
	if err != nil {
		log.Fatalf("Failed to initialize DNS server: %v", err)
	}

	serverFailures := make(chan error, 2)
	go func() {
		log.Infof("Starting DNS server on %s:%d", cfg.Server.DNSHost, cfg.Server.DNSPort)
		if err := dnsServer.Start(); err != nil {
			serverFailures <- fmt.Errorf("DNS server failed: %w", err)
		}
	}()

	// Initialize and start API server (pass dnsServer for cache clearing)
	apiServer := api.NewServer(cfg, db, filterEngine, dnsServer)

	go func() {
		log.Infof("Starting web dashboard on %s:%d", cfg.Server.APIHost, cfg.Server.APIPort)
		if err := apiServer.Start(); err != nil {
			serverFailures <- fmt.Errorf("API server failed: %w", err)
		}
	}()

	log.Info("DNS Filter is running. Press Ctrl+C to stop.")
	log.Infof("Web Dashboard: http://%s:%d", cfg.Server.APIHost, cfg.Server.APIPort)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	exitCode := 0
	select {
	case <-sigChan:
	case err := <-serverFailures:
		log.Errorf("%v", err)
		exitCode = 1
	}

	log.Info("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := dnsServer.Shutdown(ctx); err != nil {
		log.Errorf("DNS server shutdown error: %v", err)
	}
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Errorf("API server shutdown error: %v", err)
	}

	log.Info("Shutdown complete. Goodbye!")
	if exitCode != 0 {
		// The scheduled task's restart policy retries only non-zero exits.
		_ = db.Close()
		os.Exit(exitCode)
	}
}

func runAppNetworkReset() {
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		fmt.Printf("Failed to open application network state: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()
	manager, err := appnetwork.New(db.GetDB())
	if err != nil {
		fmt.Printf("Failed to initialize application network control: %v\n", err)
		os.Exit(1)
	}
	if err := manager.ResetAll(); err != nil {
		fmt.Printf("Failed to release application network rules: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Application network rules released.")
}

func hasArgument(want string) bool {
	for _, argument := range os.Args[1:] {
		if argument == want {
			return true
		}
	}
	return false
}

// setBootstrapDNS sets the Go default resolver to use Google DNS (8.8.8.8)
// This fixes the chicken-and-egg problem: DNS Filter can't download
// blocklists if it is its own DNS server and no blocklists are loaded yet.
func setBootstrapDNS() {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 10 * time.Second}
			return d.DialContext(ctx, "udp", "8.8.8.8:53")
		},
	}
}

// runDNSCommand points the OS resolver at (or away from) the local filter.
func runDNSCommand(action string) {
	switch action {
	case "setup":
		fmt.Printf("Pointing system DNS at the local filter (%s)...\n", netdns.LocalDNS)
		if err := netdns.SetDNS(); err != nil {
			fmt.Printf("✗ Failed: %v\n", err)
			fmt.Println("  Tip: run this from an Administrator (Windows) / root (Linux/macOS) shell.")
			os.Exit(1)
		}
		if err := persistDNSRoutingIntent(true); err != nil {
			_ = netdns.ResetDNS()
			fmt.Printf("✗ DNS was changed but the preference could not be saved: %v\n", err)
			fmt.Println("  The DNS route was restored to avoid leaving a partial setup.")
			os.Exit(1)
		}
		fmt.Println("✓ System DNS now points to the DNS Filter.")
		fmt.Println("  Make sure the filter is running. To undo: dns-filter reset-dns")
	case "reset":
		fmt.Println("Restoring system DNS to automatic (DHCP)...")
		if err := netdns.ResetDNS(); err != nil {
			fmt.Printf("✗ Failed: %v\n", err)
			fmt.Println("  Tip: run this from an Administrator (Windows) / root (Linux/macOS) shell.")
			os.Exit(1)
		}
		if err := persistDNSRoutingIntent(false); err != nil {
			fmt.Printf("✗ DNS was reset but the preference could not be saved: %v\n", err)
			fmt.Println("  Open the dashboard once and use DNS Controls to confirm the desired state.")
			os.Exit(1)
		}
		fmt.Println("✓ System DNS restored to automatic.")
	}
}

func persistDNSRoutingIntent(enabled bool) error {
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	cfg.Server.ManageSystemDNS = enabled
	return cfg.SaveDefault()
}

// chdirToExeDir switches the working directory to the executable's folder when
// the app's assets live there (portable/installed layout). It intentionally
// does nothing when assets aren't beside the exe (e.g. `go run`, or a build/
// binary run from the project root), preserving the existing CWD.
func chdirToExeDir() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	dir := filepath.Dir(exe)
	if _, err := os.Stat(filepath.Join(dir, "configs", "config.yaml")); err == nil {
		_ = os.Chdir(dir)
	}
}

func printBanner() {
	fmt.Printf(`
╔═══════════════════════════════════════════════════╗
║                                                   ║
║           DNS CONTENT FILTER v%s             ║
║                                                   ║
║     Protect your network from harmful content    ║
║                                                   ║
╚═══════════════════════════════════════════════════╝
`, appversion.Current)
}
