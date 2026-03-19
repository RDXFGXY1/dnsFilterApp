package gamification

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ════════════════════════════════════════════════════════════════
//  GAMIFICATION SYSTEM - DNS Filter v2.5
//  Complete gamification engine with points, levels, achievements
// ════════════════════════════════════════════════════════════════

// ── Data Structures ──────────────────────────────────────────────

type UserProfile struct {
	UserID          string    `json:"user_id"`
	Username        string    `json:"username"`
	DeviceMAC       string    `json:"device_mac"`
	Points          int       `json:"points"`
	Level           int       `json:"level"`
	Rank            string    `json:"rank"`
	CurrentStreak   int       `json:"current_streak"`
	LongestStreak   int       `json:"longest_streak"`
	TotalBlocked    int       `json:"total_blocked"`
	TotalAllowed    int       `json:"total_allowed"`
	BlockSuccessRate float64 `json:"block_success_rate"`
	CreatedAt       time.Time `json:"created_at"`
	LastActive      time.Time `json:"last_active"`
	Avatar          string    `json:"avatar"`
	Title           string    `json:"title"`
}

type Achievement struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Category    string    `json:"category"`
	Points      int       `json:"points"`
	Rarity      string    `json:"rarity"` // common, rare, epic, legendary
	UnlockedAt  *time.Time `json:"unlocked_at,omitempty"`
	Progress    int       `json:"progress"`
	Target      int       `json:"target"`
}

type Challenge struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // daily, weekly, special
	Target      int       `json:"target"`
	Progress    int       `json:"progress"`
	Reward      int       `json:"reward"`
	ExpiresAt   time.Time `json:"expires_at"`
	Completed   bool      `json:"completed"`
}

type LeaderboardEntry struct {
	Rank     int    `json:"rank"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Points   int    `json:"points"`
	Level    int    `json:"level"`
	Avatar   string `json:"avatar"`
	Streak   int    `json:"streak"`
}

type Reward struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Cost        int    `json:"cost"`
	Type        string `json:"type"` // whitelist_temp, extra_time, custom_avatar
	Duration    int    `json:"duration"` // minutes
}

// ── Gamification Engine ──────────────────────────────────────────

type Engine struct {
	db              *sql.DB
	mu              sync.RWMutex
	achievements    map[string]*Achievement
	challenges      map[string]*Challenge
	pointsPerBlock  int
	pointsPerDay    int
}

func NewEngine(db *sql.DB) (*Engine, error) {
	engine := &Engine{
		db:              db,
		achievements:    make(map[string]*Achievement),
		challenges:      make(map[string]*Challenge),
		pointsPerBlock:  10,  // 10 points per successful block resist
		pointsPerDay:    50,  // 50 points for daily login
	}

	// Initialize database tables
	if err := engine.initTables(); err != nil {
		return nil, err
	}

	// Load achievements
	engine.loadAchievements()

	// Generate daily challenges
	engine.generateDailyChallenges()

	return engine, nil
}

// ── Database Schema ──────────────────────────────────────────────

func (e *Engine) initTables() error {
	queries := []string{
		// User profiles
		`CREATE TABLE IF NOT EXISTS gamification_users (
			user_id TEXT PRIMARY KEY,
			username TEXT NOT NULL,
			device_mac TEXT,
			points INTEGER DEFAULT 0,
			level INTEGER DEFAULT 1,
			rank TEXT DEFAULT 'Bronze',
			current_streak INTEGER DEFAULT 0,
			longest_streak INTEGER DEFAULT 0,
			total_blocked INTEGER DEFAULT 0,
			total_allowed INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_active TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			avatar TEXT DEFAULT 'default',
			title TEXT DEFAULT 'Beginner'
		)`,

		// User achievements
		`CREATE TABLE IF NOT EXISTS gamification_achievements (
			user_id TEXT,
			achievement_id TEXT,
			unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			progress INTEGER DEFAULT 0,
			PRIMARY KEY (user_id, achievement_id)
		)`,

		// Active challenges
		`CREATE TABLE IF NOT EXISTS gamification_challenges (
			user_id TEXT,
			challenge_id TEXT,
			progress INTEGER DEFAULT 0,
			completed BOOLEAN DEFAULT FALSE,
			expires_at TIMESTAMP,
			PRIMARY KEY (user_id, challenge_id)
		)`,

		// Daily activity log
		`CREATE TABLE IF NOT EXISTS gamification_activity (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT,
			action_type TEXT,
			points INTEGER,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Rewards store
		`CREATE TABLE IF NOT EXISTS gamification_rewards (
			id TEXT PRIMARY KEY,
			name TEXT,
			description TEXT,
			cost INTEGER,
			type TEXT,
			duration INTEGER
		)`,

		// Purchased rewards
		`CREATE TABLE IF NOT EXISTS gamification_purchases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT,
			reward_id TEXT,
			purchased_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			used BOOLEAN DEFAULT FALSE
		)`,
	}

	for _, query := range queries {
		if _, err := e.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	return nil
}

// ── Points System ────────────────────────────────────────────────

func (e *Engine) AddPoints(userID string, points int, reason string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Update user points
	_, err := e.db.Exec(`
		UPDATE gamification_users 
		SET points = points + ?,
		    last_active = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, points, userID)

	if err != nil {
		return err
	}

	// Log activity
	_, err = e.db.Exec(`
		INSERT INTO gamification_activity (user_id, action_type, points)
		VALUES (?, ?, ?)
	`, userID, reason, points)

	// Check level up
	e.checkLevelUp(userID)

	// Check achievements
	e.checkAchievements(userID)

	return err
}

// ── Level System ─────────────────────────────────────────────────

func (e *Engine) checkLevelUp(userID string) {
	var points, level int
	e.db.QueryRow(`
		SELECT points, level FROM gamification_users WHERE user_id = ?
	`, userID).Scan(&points, &level)

	// Level formula: Level = sqrt(points / 100)
	// Level 1: 0-99 points
	// Level 2: 100-399 points
	// Level 3: 400-899 points
	// etc.
	newLevel := 1 + int(float64(points)/100)

	if newLevel > level {
		// Level up!
		rank := e.getRankForLevel(newLevel)
		e.db.Exec(`
			UPDATE gamification_users 
			SET level = ?, rank = ?
			WHERE user_id = ?
		`, newLevel, rank, userID)

		// Award level up bonus
		bonus := newLevel * 50
		e.AddPoints(userID, bonus, "level_up_bonus")
	}
}

func (e *Engine) getRankForLevel(level int) string {
	if level >= 100 {
		return "Legendary Master"
	} else if level >= 75 {
		return "Grand Master"
	} else if level >= 50 {
		return "Master"
	} else if level >= 40 {
		return "Diamond"
	} else if level >= 30 {
		return "Platinum"
	} else if level >= 20 {
		return "Gold"
	} else if level >= 10 {
		return "Silver"
	}
	return "Bronze"
}

// ── Streak System ────────────────────────────────────────────────

func (e *Engine) UpdateStreak(userID string) error {
	var lastActive time.Time
	var currentStreak, longestStreak int

	err := e.db.QueryRow(`
		SELECT last_active, current_streak, longest_streak 
		FROM gamification_users 
		WHERE user_id = ?
	`, userID).Scan(&lastActive, &currentStreak, &longestStreak)

	if err != nil {
		return err
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)

	// Check if active yesterday
	if lastActive.Format("2006-01-02") == yesterday.Format("2006-01-02") {
		// Continue streak
		currentStreak++
	} else if lastActive.Format("2006-01-02") != now.Format("2006-01-02") {
		// Streak broken
		currentStreak = 1
	}

	// Update longest streak
	if currentStreak > longestStreak {
		longestStreak = currentStreak
	}

	// Update database
	_, err = e.db.Exec(`
		UPDATE gamification_users 
		SET current_streak = ?,
		    longest_streak = ?,
		    last_active = CURRENT_TIMESTAMP
		WHERE user_id = ?
	`, currentStreak, longestStreak, userID)

	// Award streak bonus
	if currentStreak >= 7 {
		bonus := currentStreak * 10
		e.AddPoints(userID, bonus, "streak_bonus")
	}

	return err
}

// ── Achievements System ──────────────────────────────────────────

func (e *Engine) loadAchievements() {
	achievements := []Achievement{
		// Beginner achievements
		{"first_block", "First Step", "Block your first domain", "🛡️", "starter", 10, "common", nil, 0, 1},
		{"ten_blocks", "Getting Started", "Block 10 domains", "⚔️", "starter", 50, "common", nil, 0, 10},
		{"hundred_blocks", "Defender", "Block 100 domains", "🏰", "progress", 200, "rare", nil, 0, 100},
		{"thousand_blocks", "Guardian", "Block 1,000 domains", "👑", "progress", 1000, "epic", nil, 0, 1000},
		{"ten_thousand_blocks", "Legend", "Block 10,000 domains", "⭐", "progress", 5000, "legendary", nil, 0, 10000},

		// Streak achievements
		{"streak_7", "Week Warrior", "7-day streak", "🔥", "streak", 100, "common", nil, 0, 7},
		{"streak_30", "Month Master", "30-day streak", "💪", "streak", 500, "rare", nil, 0, 30},
		{"streak_100", "Century Champion", "100-day streak", "🏆", "streak", 2000, "epic", nil, 0, 100},
		{"streak_365", "Year Legend", "365-day streak", "👑", "streak", 10000, "legendary", nil, 0, 365},

		// Ramadan achievements
		{"ramadan_day_1", "Ramadan Begins", "First day of Ramadan", "🌙", "ramadan", 100, "common", nil, 0, 1},
		{"ramadan_complete", "Ramadan Master", "Complete entire Ramadan", "🕌", "ramadan", 5000, "legendary", nil, 0, 30},
		{"all_prayers", "Prayer Warrior", "All 5 prayers on time for a day", "🤲", "ramadan", 200, "rare", nil, 0, 5},

		// Time achievements
		{"early_bird", "Early Bird", "Block site before 6 AM", "🌅", "time", 50, "common", nil, 0, 1},
		{"night_owl", "Night Guard", "Block site after midnight", "🌙", "time", 50, "common", nil, 0, 1},

		// Social achievements
		{"invite_friend", "Social Butterfly", "Invite a friend", "🦋", "social", 200, "rare", nil, 0, 1},
		{"family_leader", "Family Leader", "Top of family leaderboard", "👨‍👩‍👧‍👦", "social", 500, "epic", nil, 0, 1},

		// Special achievements
		{"perfect_day", "Perfect Day", "No blocked attempts for 24 hours", "✨", "special", 300, "rare", nil, 0, 1},
		{"weekend_warrior", "Weekend Warrior", "Perfect weekend (no blocked attempts)", "🎯", "special", 500, "epic", nil, 0, 1},
		{"self_control", "Self Control Master", "Resist blocking same site 10 times", "🧘", "special", 1000, "legendary", nil, 0, 10},
	}

	for _, achievement := range achievements {
		e.achievements[achievement.ID] = &achievement
	}
}

func (e *Engine) checkAchievements(userID string) {
	var totalBlocked, currentStreak int
	e.db.QueryRow(`
		SELECT total_blocked, current_streak 
		FROM gamification_users 
		WHERE user_id = ?
	`, userID).Scan(&totalBlocked, &currentStreak)

	// Check each achievement
	for _, achievement := range e.achievements {
		// Skip if already unlocked
		var unlocked bool
		e.db.QueryRow(`
			SELECT COUNT(*) > 0 
			FROM gamification_achievements 
			WHERE user_id = ? AND achievement_id = ?
		`, userID, achievement.ID).Scan(&unlocked)

		if unlocked {
			continue
		}

		// Check progress
		progress := 0
		switch achievement.Category {
		case "starter", "progress":
			progress = totalBlocked
		case "streak":
			progress = currentStreak
		}

		// Update progress
		e.db.Exec(`
			INSERT OR REPLACE INTO gamification_achievements 
			(user_id, achievement_id, progress)
			VALUES (?, ?, ?)
		`, userID, achievement.ID, progress)

		// Check if unlocked
		if progress >= achievement.Target {
			e.unlockAchievement(userID, achievement.ID)
		}
	}
}

func (e *Engine) unlockAchievement(userID, achievementID string) {
	achievement := e.achievements[achievementID]
	
	// Mark as unlocked
	e.db.Exec(`
		UPDATE gamification_achievements 
		SET unlocked_at = CURRENT_TIMESTAMP 
		WHERE user_id = ? AND achievement_id = ?
	`, userID, achievementID)

	// Award points
	e.AddPoints(userID, achievement.Points, fmt.Sprintf("achievement_%s", achievementID))
}

// ── Challenge System ─────────────────────────────────────────────

func (e *Engine) generateDailyChallenges() {
	challenges := []Challenge{
		{
			ID:          "daily_block_10",
			Name:        "Daily Defender",
			Description: "Block 10 domains today",
			Type:        "daily",
			Target:      10,
			Reward:      100,
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		},
		{
			ID:          "daily_no_social",
			Name:        "Social Detox",
			Description: "Don't access social media today",
			Type:        "daily",
			Target:      0, // 0 social media accesses
			Reward:      200,
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		},
		{
			ID:          "daily_perfect",
			Name:        "Perfect Day",
			Description: "No blocked attempts all day",
			Type:        "daily",
			Target:      0,
			Reward:      300,
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		},
	}

	for _, challenge := range challenges {
		e.challenges[challenge.ID] = &challenge
	}
}

func (e *Engine) UpdateChallenge(userID, challengeID string, progress int) {
	e.db.Exec(`
		INSERT OR REPLACE INTO gamification_challenges 
		(user_id, challenge_id, progress, expires_at)
		VALUES (?, ?, ?, ?)
	`, userID, challengeID, progress, e.challenges[challengeID].ExpiresAt)

	// Check if completed
	if progress >= e.challenges[challengeID].Target {
		e.completeChallenge(userID, challengeID)
	}
}

func (e *Engine) completeChallenge(userID, challengeID string) {
	e.db.Exec(`
		UPDATE gamification_challenges 
		SET completed = TRUE 
		WHERE user_id = ? AND challenge_id = ?
	`, userID, challengeID)

	// Award reward
	reward := e.challenges[challengeID].Reward
	e.AddPoints(userID, reward, fmt.Sprintf("challenge_%s", challengeID))
}

// ── Leaderboard ──────────────────────────────────────────────────

func (e *Engine) GetLeaderboard(limit int) ([]LeaderboardEntry, error) {
	rows, err := e.db.Query(`
		SELECT user_id, username, points, level, avatar, current_streak
		FROM gamification_users
		ORDER BY points DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leaderboard []LeaderboardEntry
	rank := 1

	for rows.Next() {
		var entry LeaderboardEntry
		rows.Scan(&entry.UserID, &entry.Username, &entry.Points, 
		          &entry.Level, &entry.Avatar, &entry.Streak)
		entry.Rank = rank
		rank++
		leaderboard = append(leaderboard, entry)
	}

	return leaderboard, nil
}

// ── User Profile ─────────────────────────────────────────────────

func (e *Engine) GetUserProfile(userID string) (*UserProfile, error) {
	var profile UserProfile
	err := e.db.QueryRow(`
		SELECT user_id, username, device_mac, points, level, rank,
		       current_streak, longest_streak, total_blocked, total_allowed,
		       created_at, last_active, avatar, title
		FROM gamification_users
		WHERE user_id = ?
	`, userID).Scan(
		&profile.UserID, &profile.Username, &profile.DeviceMAC,
		&profile.Points, &profile.Level, &profile.Rank,
		&profile.CurrentStreak, &profile.LongestStreak,
		&profile.TotalBlocked, &profile.TotalAllowed,
		&profile.CreatedAt, &profile.LastActive,
		&profile.Avatar, &profile.Title,
	)

	if err != nil {
		return nil, err
	}

	// Calculate success rate
	total := profile.TotalBlocked + profile.TotalAllowed
	if total > 0 {
		profile.BlockSuccessRate = float64(profile.TotalBlocked) / float64(total) * 100
	}

	return &profile, nil
}

func (e *Engine) CreateUser(userID, username, deviceMAC string) error {
	_, err := e.db.Exec(`
		INSERT INTO gamification_users (user_id, username, device_mac)
		VALUES (?, ?, ?)
	`, userID, username, deviceMAC)
	return err
}

// ── Rewards Store ────────────────────────────────────────────────

func (e *Engine) GetRewards() []Reward {
	return []Reward{
		{"temp_whitelist_30", "30 Minutes Freedom", "Unblock any site for 30 minutes", 200, "whitelist_temp", 30},
		{"temp_whitelist_60", "1 Hour Freedom", "Unblock any site for 1 hour", 350, "whitelist_temp", 60},
		{"extra_time", "Extra Screen Time", "Add 30 minutes to daily limit", 150, "extra_time", 30},
		{"custom_avatar", "Custom Avatar", "Unlock custom avatar", 500, "custom_avatar", 0},
		{"theme_unlock", "Premium Theme", "Unlock premium dashboard theme", 1000, "theme", 0},
	}
}

func (e *Engine) PurchaseReward(userID, rewardID string) error {
	// Get reward details
	var cost int
	for _, reward := range e.GetRewards() {
		if reward.ID == rewardID {
			cost = reward.Cost
			break
		}
	}

	// Check if user has enough points
	var points int
	e.db.QueryRow(`SELECT points FROM gamification_users WHERE user_id = ?`, userID).Scan(&points)

	if points < cost {
		return fmt.Errorf("insufficient points")
	}

	// Deduct points
	e.db.Exec(`UPDATE gamification_users SET points = points - ? WHERE user_id = ?`, cost, userID)

	// Add purchase record
	expiresAt := time.Now().Add(24 * time.Hour) // Default 24 hours
	e.db.Exec(`
		INSERT INTO gamification_purchases (user_id, reward_id, expires_at)
		VALUES (?, ?, ?)
	`, userID, rewardID, expiresAt)

	return nil
}

// ── Event Tracking ───────────────────────────────────────────────

func (e *Engine) OnBlockAttempt(userID string, domain string, blocked bool) {
	if blocked {
		// Increment total blocked
		e.db.Exec(`UPDATE gamification_users SET total_blocked = total_blocked + 1 WHERE user_id = ?`, userID)
		
		// Award points for resisting temptation
		e.AddPoints(userID, e.pointsPerBlock, "blocked_attempt")
	} else {
		// Increment allowed
		e.db.Exec(`UPDATE gamification_users SET total_allowed = total_allowed + 1 WHERE user_id = ?`, userID)
	}

	// Update streak
	e.UpdateStreak(userID)
}

func (e *Engine) OnDailyLogin(userID string) {
	e.AddPoints(userID, e.pointsPerDay, "daily_login")
	e.UpdateStreak(userID)
}

// ── Export/Import ────────────────────────────────────────────────

func (e *Engine) ExportUserData(userID string) ([]byte, error) {
	profile, _ := e.GetUserProfile(userID)
	return json.MarshalIndent(profile, "", "  ")
}
