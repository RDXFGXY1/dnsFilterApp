package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RDXFGXY1/dns-filter-app/internal/api"
	"github.com/RDXFGXY1/dns-filter-app/internal/config"
	"github.com/RDXFGXY1/dns-filter-app/internal/database"
	"github.com/RDXFGXY1/dns-filter-app/internal/dns"
	"github.com/RDXFGXY1/dns-filter-app/internal/filter"
	"github.com/RDXFGXY1/dns-filter-app/pkg/logger"
)

var (
	configPath = flag.String("config", "./configs/config.yaml", "Path to configuration file")
	devMode    = flag.Bool("dev", false, "Run in development mode")
	version    = "2.0.0"
)

func main() {
	flag.Parse()

	printBanner()

	// Initialize logger
	log := logger.New(*devMode)
	log.Infof("Starting DNS Filter Application v%s", version)

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

	go func() {
		log.Infof("Starting DNS server on %s:%d", cfg.Server.DNSHost, cfg.Server.DNSPort)
		if err := dnsServer.Start(); err != nil {
			log.Fatalf("DNS server failed: %v", err)
		}
	}()

	// Initialize and start API server (pass dnsServer for cache clearing)
	apiServer := api.NewServer(cfg, db, filterEngine, dnsServer)

	go func() {
		log.Infof("Starting web dashboard on %s:%d", cfg.Server.APIHost, cfg.Server.APIPort)
		if err := apiServer.Start(); err != nil {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	log.Info("DNS Filter is running. Press Ctrl+C to stop.")
	log.Infof("Web Dashboard: http://%s:%d", cfg.Server.APIHost, cfg.Server.APIPort)

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

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

func printBanner() {
	fmt.Printf(`
╔═══════════════════════════════════════════════════╗
║                                                   ║
║           DNS CONTENT FILTER v%s             ║
║                                                   ║
║     Protect your network from harmful content    ║
║                                                   ║
╚═══════════════════════════════════════════════════╝
`, version)
}
