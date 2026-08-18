package blockpage

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"
)

// ════════════════════════════════════════════════════════════════
//  CUSTOM BLOCK PAGE - DNS Filter v2.5
//  Beautiful block page instead of NXDOMAIN
// ════════════════════════════════════════════════════════════════

type BlockPageServer struct {
	server   *http.Server
	db       *sql.DB
	template *template.Template
	stats    *BlockStats
	mu       sync.RWMutex

	handler http.Handler // shared by the HTTP and HTTPS servers
	tlsMgr  *tlsManager  // dynamic per-SNI certificate authority (nil until EnableTLS)
}

type BlockStats struct {
	TotalBlocks      int
	BlocksToday      int
	TimeSavedMinutes int
	TopBlockedDomain string
}

type BlockPageData struct {
	Domain            string
	Reason            string
	Category          string
	Keywords          []string
	Timestamp         string
	UserAgent         string
	ClientIP          string
	BlockedToday      int
	TotalBlocks       int
	TimeSaved         int
	Quote             string
	ShowRamadanMode   bool
	CanRequestUnblock bool
	UnblockMinutes    int
}

func NewBlockPageServer(db *sql.DB, port int) (*BlockPageServer, error) {
	bps := &BlockPageServer{
		db:    db,
		stats: &BlockStats{},
	}

	// Initialize database tables
	if err := bps.initTables(); err != nil {
		return nil, err
	}

	// Load template
	if err := bps.loadTemplate(); err != nil {
		return nil, err
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", bps.handleBlockPage)
	mux.HandleFunc("/request-unblock", bps.handleUnblockRequest)
	mux.HandleFunc("/stats", bps.handleStats)

	bps.handler = mux
	bps.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	return bps, nil
}

// ── Database Schema ──────────────────────────────────────────────

func (bps *BlockPageServer) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS block_page_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT,
			reason TEXT,
			category TEXT,
			client_ip TEXT,
			user_agent TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS unblock_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT,
			client_ip TEXT,
			reason TEXT,
			status TEXT DEFAULT 'pending',
			requested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			approved_at TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := bps.db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

// ── Template ─────────────────────────────────────────────────────

func (bps *BlockPageServer) loadTemplate() error {
	tmpl := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>🛡️ Site Blocked - DNS Filter</title>
    <style>
        :root {
            --bg: #f6f1e7; --surface: #fffdf9; --surface-2: #f3ecdf;
            --ink: #1a1a1a; --muted: #6b6558; --faint: #9a9384;
            --coral: #d1553b; --coral-soft: #f7e2dc; --green: #2f8a55;
        }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            background: var(--bg); color: var(--ink);
            font-family: "JetBrains Mono", "Cascadia Code", "Segoe UI Mono", Consolas, monospace;
            min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 24px;
        }
        .container {
            max-width: 640px; width: 100%;
            background: var(--surface); border: 2px solid var(--ink);
            box-shadow: 8px 8px 0 var(--ink); padding: 40px;
        }
        .header { text-align: center; margin-bottom: 28px; }
        .shield {
            display: inline-grid; place-items: center; width: 64px; height: 64px;
            background: var(--coral); border: 2px solid var(--ink); box-shadow: 3px 3px 0 var(--ink);
            font-size: 34px; margin-bottom: 16px;
        }
        h1 {
            color: var(--coral); font-size: 2.2em; letter-spacing: -.02em;
            text-transform: uppercase; margin-bottom: 6px;
        }
        .header p { color: var(--muted); text-transform: uppercase; letter-spacing: .08em; font-size: .8em; }
        .domain {
            background: var(--coral-soft); border: 1.5px solid var(--ink);
            padding: 14px 16px; margin: 22px 0; word-break: break-all; font-size: 1.05em;
        }
        .domain strong { color: var(--coral); }
        .reason-box {
            background: var(--surface-2); border: 1.5px solid var(--ink); border-left: 6px solid var(--coral);
            padding: 18px 20px; margin: 20px 0;
        }
        .reason-title { font-weight: 700; margin-bottom: 8px; text-transform: uppercase; letter-spacing: .02em; }
        .stats-grid {
            display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 14px; margin: 26px 0;
        }
        .stat-card {
            background: var(--surface-2); border: 1.5px solid var(--ink); box-shadow: 2px 2px 0 var(--ink);
            padding: 16px; text-align: center;
        }
        .stat-number { font-size: 1.9em; font-weight: 700; color: var(--coral); display: block; letter-spacing: -.02em; }
        .stat-label { font-size: .72em; color: var(--muted); margin-top: 6px; text-transform: uppercase; letter-spacing: .04em; }
        .quote-box {
            background: var(--surface-2); border: 1.5px solid var(--ink); border-left: 6px solid var(--green);
            padding: 16px 20px; margin: 26px 0; font-style: italic;
        }
        .buttons { display: flex; gap: 14px; margin-top: 28px; flex-wrap: wrap; }
        .btn {
            flex: 1; min-width: 200px; padding: 14px 24px; cursor: pointer;
            border: 2px solid var(--ink); box-shadow: 3px 3px 0 var(--ink);
            background: var(--coral); color: #fff;
            font-family: inherit; font-size: .95em; font-weight: 700; text-transform: uppercase; letter-spacing: .03em;
            text-decoration: none; display: inline-block; text-align: center;
            transition: transform .08s, box-shadow .08s;
        }
        .btn:hover { transform: translate(-1px,-1px); box-shadow: 4px 4px 0 var(--ink); }
        .btn:active { transform: translate(2px,2px); box-shadow: 1px 1px 0 var(--ink); }
        .btn-secondary { background: var(--surface); color: var(--ink); }
        .keywords { display: flex; flex-wrap: wrap; gap: 8px; margin-top: 10px; }
        .keyword-tag { background: var(--coral-soft); border: 1.5px solid var(--ink); padding: 4px 12px; font-size: .85em; }
        .footer { text-align: center; margin-top: 36px; padding-top: 18px; border-top: 2px solid var(--ink); color: var(--muted); font-size: .85em; }
        .footer a { color: var(--coral); }
        .timestamp { color: var(--faint); font-size: .82em; text-align: center; margin-top: 18px; }
        .ramadan-mode { border-color: var(--ink); }
        @media (max-width: 600px) {
            .container { padding: 24px; }
            h1 { font-size: 1.7em; }
            .buttons { flex-direction: column; }
            .btn { width: 100%; }
        }
    </style>
</head>
<body>
    <div class="container{{if .ShowRamadanMode}} ramadan-mode{{end}}">
        <div class="header">
            <div class="shield">🛡️</div>
            <h1>SITE BLOCKED</h1>
            <p>DNS Content Filter</p>
        </div>

        <div class="domain">
            <strong>Domain:</strong> {{.Domain}}
        </div>

        <div class="reason-box">
            <div class="reason-title">🚫 Reason for blocking:</div>
            <p>{{.Reason}}</p>
            {{if .Category}}
            <p style="margin-top: 10px;"><strong>Category:</strong> {{.Category}}</p>
            {{end}}
            {{if .Keywords}}
            <div style="margin-top: 15px;">
                <strong>Matched Keywords:</strong>
                <div class="keywords">
                    {{range .Keywords}}
                    <span class="keyword-tag">{{.}}</span>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <span class="stat-number">{{.BlockedToday}}</span>
                <span class="stat-label">Blocked Today</span>
            </div>
            <div class="stat-card">
                <span class="stat-number">{{.TotalBlocks}}</span>
                <span class="stat-label">Total Blocks</span>
            </div>
            <div class="stat-card">
                <span class="stat-number">{{.TimeSaved}}</span>
                <span class="stat-label">Minutes Saved</span>
            </div>
        </div>

        {{if .Quote}}
        <div class="quote-box">
            💭 {{.Quote}}
        </div>
        {{end}}

        <div class="buttons">
            <a href="javascript:history.back()" class="btn">← Go Back</a>
            {{if .CanRequestUnblock}}
            <button onclick="requestUnblock()" class="btn btn-secondary">Request Unblock</button>
            {{end}}
        </div>

        <div class="timestamp">
            Blocked at: {{.Timestamp}}
        </div>

        <div class="footer">
            <p>🔒 Protected by DNS Filter v2.6</p>
            <p>Dashboard: <a href="http://127.0.0.1:8080">127.0.0.1:8080</a></p>
        </div>
    </div>

    <script>
        function requestUnblock() {
            const domain = '{{.Domain}}';
            const reason = prompt('Why should this site be unblocked?');
            
            if (reason && reason.trim()) {
                fetch('/request-unblock', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        domain: domain,
                        reason: reason
                    })
                })
                .then(response => response.json())
                .then(data => {
                    alert(data.message || 'Request submitted! An admin will review it.');
                })
                .catch(() => {
                    alert('Failed to submit request. Please try again.');
                });
            }
        }
    </script>
</body>
</html>`

	var err error
	bps.template, err = template.New("blockpage").Parse(tmpl)
	return err
}

// ── HTTP Handlers ────────────────────────────────────────────────

func (bps *BlockPageServer) handleBlockPage(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	domain := r.URL.Query().Get("domain")
	reason := r.URL.Query().Get("reason")
	category := r.URL.Query().Get("category")

	if domain == "" {
		domain = r.Host
	}

	// SAFETY CHECK: If domain is whitelisted, don't show block page
	// This handles the case where unblocking takes effect but browser
	// still connects to 127.0.0.1 due to connection pooling
	if bps.isWhitelisted(domain) {
		// Redirect to actual domain instead of showing block page
		scheme := "http"
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		w.Header().Set("Location", scheme+"://"+domain)
		w.WriteHeader(http.StatusMovedPermanently)
		return
	}

	// Log block
	bps.logBlock(domain, reason, category, r.RemoteAddr, r.UserAgent())

	// Update stats
	bps.updateStats()

	// Prepare data
	data := BlockPageData{
		Domain:            domain,
		Reason:            bps.formatReason(reason),
		Category:          category,
		Timestamp:         time.Now().Format("2006-01-02 15:04:05"),
		ClientIP:          r.RemoteAddr,
		UserAgent:         r.UserAgent(),
		BlockedToday:      bps.stats.BlocksToday,
		TotalBlocks:       bps.stats.TotalBlocks,
		TimeSaved:         bps.stats.TimeSavedMinutes,
		Quote:             bps.getRandomQuote(),
		ShowRamadanMode:   bps.isRamadanMode(),
		CanRequestUnblock: true,
		UnblockMinutes:    30,
	}

	// Extract keywords if in reason
	if keywords := r.URL.Query().Get("keywords"); keywords != "" {
		// Parse keywords from comma-separated string
		data.Keywords = []string{keywords}
	}

	// Render template
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	bps.template.Execute(w, data)
}

func (bps *BlockPageServer) handleUnblockRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request (simplified - should decode JSON properly)
	domain := r.FormValue("domain")
	reason := r.FormValue("reason")
	clientIP := r.RemoteAddr

	// Save request
	_, err := bps.db.Exec(`
		INSERT INTO unblock_requests (domain, client_ip, reason)
		VALUES (?, ?, ?)
	`, domain, clientIP, reason)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"success": false, "message": "Failed to save request"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success": true, "message": "Request submitted for admin review"}`))
}

func (bps *BlockPageServer) handleStats(w http.ResponseWriter, r *http.Request) {
	bps.mu.RLock()
	defer bps.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"total_blocks": %d,
		"blocks_today": %d,
		"time_saved_minutes": %d,
		"top_blocked_domain": "%s"
	}`, bps.stats.TotalBlocks, bps.stats.BlocksToday,
		bps.stats.TimeSavedMinutes, bps.stats.TopBlockedDomain)
}

// ── Helper Functions ─────────────────────────────────────────────

func (bps *BlockPageServer) logBlock(domain, reason, category, clientIP, userAgent string) {
	bps.db.Exec(`
		INSERT INTO block_page_logs (domain, reason, category, client_ip, user_agent)
		VALUES (?, ?, ?, ?, ?)
	`, domain, reason, category, clientIP, userAgent)
}

func (bps *BlockPageServer) updateStats() {
	bps.mu.Lock()
	defer bps.mu.Unlock()

	// Total blocks
	bps.db.QueryRow("SELECT COUNT(*) FROM block_page_logs").Scan(&bps.stats.TotalBlocks)

	// Blocks today
	bps.db.QueryRow(`
		SELECT COUNT(*) FROM block_page_logs 
		WHERE DATE(timestamp) = DATE('now')
	`).Scan(&bps.stats.BlocksToday)

	// Estimate time saved (2 minutes per block)
	bps.stats.TimeSavedMinutes = bps.stats.TotalBlocks * 2

	// Top blocked domain
	bps.db.QueryRow(`
		SELECT domain FROM block_page_logs 
		GROUP BY domain 
		ORDER BY COUNT(*) DESC 
		LIMIT 1
	`).Scan(&bps.stats.TopBlockedDomain)
}

func (bps *BlockPageServer) formatReason(reason string) string {
	reasons := map[string]string{
		"category:adult":    "This site belongs to the Adult Content category",
		"category:social":   "Social media sites are currently blocked",
		"category:gaming":   "Gaming sites are currently blocked",
		"category:gambling": "Gambling and betting sites are blocked",
		"keyword":           "This domain contains blocked keywords",
		"blocklist":         "This domain is in the blocklist",
		"ai":                "AI detected harmful content patterns",
	}

	if formatted, exists := reasons[reason]; exists {
		return formatted
	}

	return "This site has been blocked by your DNS filter"
}

func (bps *BlockPageServer) getSetting(key string) string {
	var v string
	if err := bps.db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&v); err != nil {
		return ""
	}
	return v
}

func (bps *BlockPageServer) getRandomQuote() string {
	// Custom message overrides everything.
	if msg := bps.getSetting("blockpage_message"); msg != "" {
		return msg
	}

	roast := []string{
		"Nice try. This site was NOT it.",
		"Blocked. Go touch some grass. 🌱",
		"Imagine getting stopped by your own DNS. Embarrassing.",
		"That's a no from us, champ.",
		"You really thought? Blocked.",
		"Denied. Your future self says you're welcome.",
		"This ain't the app for that. Back to work.",
	}
	meme := []string{
		"🚫 This is not the website you're looking for.",
		"404: Willpower found. Site blocked.",
		"One does not simply visit this site. 🧙",
		"Task failed successfully — you've been blocked.",
		"Ain't nobody got time for that. Blocked.",
		"Sir, this is a Wendy's. (It's blocked.)",
	}
	quotes := []string{
		"Every temptation resisted is a victory won.",
		"You're building a better future, one block at a time.",
		"Focus on what matters. Stay strong.",
		"Time saved: time gained for what's important.",
		"Your future self will thank you.",
		"Discipline is choosing between what you want now and what you want most.",
		"Small choices today, big results tomorrow.",
		"You have the power to control your digital life.",
	}

	switch bps.getSetting("blockpage_mode") {
	case "roast":
		return roast[time.Now().Unix()%int64(len(roast))]
	case "meme":
		return meme[time.Now().Unix()%int64(len(meme))]
	}

	// Islamic quotes for Ramadan
	ramadanQuotes := []string{
		"وَاسْتَعِينُوا بِالصَّبْرِ وَالصَّلَاةِ - Seek help through patience and prayer",
		"Indeed, the patient will be given their reward without account. (Quran 39:10)",
		"فَإِنَّ مَعَ الْعُسْرِ يُسْرًا - Indeed, with hardship comes ease (94:5)",
		"The month of Ramadan is the month of patience and self-restraint.",
		"May Allah accept your efforts in avoiding sin.",
	}

	if bps.isRamadanMode() {
		return ramadanQuotes[time.Now().Unix()%int64(len(ramadanQuotes))]
	}

	return quotes[time.Now().Unix()%int64(len(quotes))]
}

func (bps *BlockPageServer) isRamadanMode() bool {
	// Check if Ramadan mode is enabled in config
	// For now, return false - implement config check
	return false
}

// isWhitelisted checks if a domain is in the whitelist
func (bps *BlockPageServer) isWhitelisted(domain string) bool {
	if domain == "" {
		return false
	}

	// Query the whitelist table from database
	var count int
	err := bps.db.QueryRow("SELECT COUNT(*) FROM whitelist WHERE domain = ?", domain).Scan(&count)
	if err != nil {
		return false
	}

	return count > 0
}

// ── Server Control ───────────────────────────────────────────────

func (bps *BlockPageServer) Start() error {
	go bps.server.ListenAndServe()
	return nil
}

func (bps *BlockPageServer) Stop() error {
	return bps.server.Close()
}

// ── Statistics ───────────────────────────────────────────────────

func (bps *BlockPageServer) GetUnblockRequests() ([]map[string]interface{}, error) {
	rows, err := bps.db.Query(`
		SELECT id, domain, client_ip, reason, status, requested_at
		FROM unblock_requests
		WHERE status = 'pending'
		ORDER BY requested_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []map[string]interface{}
	for rows.Next() {
		var id int
		var domain, clientIP, reason, status, requestedAt string
		rows.Scan(&id, &domain, &clientIP, &reason, &status, &requestedAt)

		requests = append(requests, map[string]interface{}{
			"id":           id,
			"domain":       domain,
			"client_ip":    clientIP,
			"reason":       reason,
			"status":       status,
			"requested_at": requestedAt,
		})
	}

	return requests, nil
}

func (bps *BlockPageServer) ApproveUnblockRequest(id int) error {
	_, err := bps.db.Exec(`
		UPDATE unblock_requests 
		SET status = 'approved', approved_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, id)
	return err
}

func (bps *BlockPageServer) RejectUnblockRequest(id int) error {
	_, err := bps.db.Exec(`
		UPDATE unblock_requests 
		SET status = 'rejected'
		WHERE id = ?
	`, id)
	return err
}
