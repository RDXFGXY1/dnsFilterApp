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
}

type BlockStats struct {
	TotalBlocks      int
	BlocksToday      int
	TimeSavedMinutes int
	TopBlockedDomain string
}

type BlockPageData struct {
	Domain           string
	Reason           string
	Category         string
	Keywords         []string
	Timestamp        string
	UserAgent        string
	ClientIP         string
	BlockedToday     int
	TotalBlocks      int
	TimeSaved        int
	Quote            string
	ShowRamadanMode  bool
	CanRequestUnblock bool
	UnblockMinutes   int
}

func NewBlockPageServer(db *sql.DB, port int) (*BlockPageServer, error) {
	bps := &BlockPageServer{
		db: db,
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
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            background: linear-gradient(135deg, #0a0a0a 0%, #1a1a1a 100%);
            color: #00ff00;
            font-family: 'Courier New', monospace;
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }

        .container {
            max-width: 800px;
            width: 100%;
            background: rgba(0, 0, 0, 0.8);
            border: 2px solid #00ff00;
            border-radius: 10px;
            box-shadow: 0 0 30px rgba(0, 255, 0, 0.3);
            padding: 40px;
            animation: slideIn 0.5s ease-out;
        }

        @keyframes slideIn {
            from {
                opacity: 0;
                transform: translateY(-30px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .header {
            text-align: center;
            margin-bottom: 30px;
        }

        .shield {
            font-size: 80px;
            animation: pulse 2s infinite;
        }

        @keyframes pulse {
            0%, 100% { transform: scale(1); }
            50% { transform: scale(1.1); }
        }

        h1 {
            color: #ff0000;
            font-size: 2.5em;
            margin: 20px 0;
            text-shadow: 0 0 10px rgba(255, 0, 0, 0.5);
        }

        .domain {
            background: rgba(255, 0, 0, 0.1);
            border: 1px solid #ff0000;
            padding: 15px;
            border-radius: 5px;
            margin: 20px 0;
            word-break: break-all;
            font-size: 1.2em;
            color: #ff6666;
        }

        .reason-box {
            background: rgba(0, 255, 0, 0.05);
            border-left: 4px solid #00ff00;
            padding: 20px;
            margin: 20px 0;
        }

        .reason-title {
            font-weight: bold;
            margin-bottom: 10px;
            font-size: 1.1em;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 15px;
            margin: 30px 0;
        }

        .stat-card {
            background: rgba(0, 255, 0, 0.05);
            border: 1px solid #00ff00;
            border-radius: 5px;
            padding: 15px;
            text-align: center;
        }

        .stat-number {
            font-size: 2em;
            font-weight: bold;
            color: #00ff00;
            display: block;
        }

        .stat-label {
            font-size: 0.9em;
            color: #888;
            margin-top: 5px;
        }

        .quote-box {
            background: rgba(0, 100, 255, 0.1);
            border-left: 4px solid #0066ff;
            padding: 20px;
            margin: 30px 0;
            font-style: italic;
            color: #66ccff;
        }

        .ramadan-mode {
            background: linear-gradient(135deg, #1a5f1a 0%, #0d4d0d 100%);
            border-color: #ffd700;
            box-shadow: 0 0 30px rgba(255, 215, 0, 0.3);
        }

        .ramadan-mode h1 {
            color: #ffd700;
            text-shadow: 0 0 10px rgba(255, 215, 0, 0.5);
        }

        .buttons {
            display: flex;
            gap: 15px;
            margin-top: 30px;
            flex-wrap: wrap;
        }

        .btn {
            flex: 1;
            min-width: 200px;
            padding: 15px 30px;
            border: 2px solid #00ff00;
            background: transparent;
            color: #00ff00;
            font-family: 'Courier New', monospace;
            font-size: 1em;
            cursor: pointer;
            border-radius: 5px;
            transition: all 0.3s;
            text-decoration: none;
            display: inline-block;
            text-align: center;
        }

        .btn:hover {
            background: #00ff00;
            color: #000;
            box-shadow: 0 0 20px rgba(0, 255, 0, 0.5);
        }

        .btn-secondary {
            border-color: #666;
            color: #666;
        }

        .btn-secondary:hover {
            background: #666;
            color: #000;
        }

        .keywords {
            display: flex;
            flex-wrap: wrap;
            gap: 10px;
            margin-top: 10px;
        }

        .keyword-tag {
            background: rgba(255, 0, 0, 0.2);
            border: 1px solid #ff0000;
            padding: 5px 15px;
            border-radius: 20px;
            font-size: 0.9em;
        }

        .footer {
            text-align: center;
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #333;
            color: #666;
            font-size: 0.9em;
        }

        .timestamp {
            color: #666;
            font-size: 0.9em;
            text-align: center;
            margin-top: 20px;
        }

        @media (max-width: 600px) {
            .container {
                padding: 20px;
            }
            
            h1 {
                font-size: 1.8em;
            }
            
            .buttons {
                flex-direction: column;
            }
            
            .btn {
                width: 100%;
            }
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
            <p>🔒 Protected by DNS Filter v2.5</p>
            <p>Dashboard: <a href="http://127.0.0.1:8080" style="color: #00ff00;">127.0.0.1:8080</a></p>
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

func (bps *BlockPageServer) getRandomQuote() string {
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
