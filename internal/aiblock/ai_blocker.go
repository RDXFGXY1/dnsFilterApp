package aiblock

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"
)

// ════════════════════════════════════════════════════════════════
//  AI-POWERED BLOCKING - DNS Filter v2.5
//  Machine learning model for intelligent blocking
// ════════════════════════════════════════════════════════════════

type AIBlocker struct {
	db      *sql.DB
	model   *NaiveBayesModel
	mu      sync.RWMutex
	enabled bool
}

type PredictionResult struct {
	Domain     string  `json:"domain"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Blocked    bool    `json:"blocked"`
	Reason     string  `json:"reason"`
	Features   map[string]float64 `json:"features"`
}

type NaiveBayesModel struct {
	vocabulary  map[string]int
	categoryCount map[string]int
	wordCategoryCount map[string]map[string]int
	totalDocs   int
}

type DomainFeatures struct {
	Length           int
	DigitCount       int
	SpecialCharCount int
	DotCount         int
	HasNumbers       bool
	HasHyphens       bool
	TLD              string
	Entropy          float64
	HasSuspiciousWords bool
}

func NewAIBlocker(db *sql.DB) (*AIBlocker, error) {
	ab := &AIBlocker{
		db:      db,
		model:   NewNaiveBayesModel(),
		enabled: true,
	}

	if err := ab.initTables(); err != nil {
		return nil, err
	}

	// Load training data
	if err := ab.loadTrainingData(); err != nil {
		return nil, err
	}

	return ab, nil
}

// ── Database Schema ──────────────────────────────────────────────

func (ab *AIBlocker) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS ai_training_data (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT UNIQUE,
			category TEXT,
			confidence REAL,
			source TEXT,
			added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS ai_predictions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT,
			category TEXT,
			confidence REAL,
			blocked BOOLEAN,
			features TEXT,
			timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS ai_false_positives (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			domain TEXT,
			predicted_category TEXT,
			actual_category TEXT,
			reported_by TEXT,
			reported_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		`CREATE TABLE IF NOT EXISTS ai_model_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			total_predictions INTEGER,
			correct_predictions INTEGER,
			false_positives INTEGER,
			accuracy REAL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, query := range queries {
		if _, err := ab.db.Exec(query); err != nil {
			return err
		}
	}

	// Initialize training data if empty
	var count int
	ab.db.QueryRow("SELECT COUNT(*) FROM ai_training_data").Scan(&count)
	
	if count == 0 {
		return ab.seedTrainingData()
	}

	return nil
}

// ── Training Data ────────────────────────────────────────────────

func (ab *AIBlocker) seedTrainingData() error {
	// Seed with known patterns
	trainingData := []struct {
		domain   string
		category string
	}{
		// Adult content patterns
		{"porn", "adult"},
		{"xxx", "adult"},
		{"sex", "adult"},
		{"nude", "adult"},
		{"adult", "adult"},
		{"cam", "adult"},
		{"tube", "adult"},
		{"xvideos", "adult"},
		{"pornhub", "adult"},
		{"xnxx", "adult"},
		
		// Gambling
		{"casino", "gambling"},
		{"bet", "gambling"},
		{"poker", "gambling"},
		{"slots", "gambling"},
		{"gambling", "gambling"},
		
		// Safe sites
		{"google", "safe"},
		{"wikipedia", "safe"},
		{"github", "safe"},
		{"stackoverflow", "safe"},
		{"amazon", "safe"},
		{"news", "safe"},
		{"education", "safe"},
		{"university", "safe"},
		
		// Malware/suspicious
		{"free", "suspicious"},
		{"crack", "suspicious"},
		{"download", "suspicious"},
		{"torrent", "suspicious"},
		{"warez", "suspicious"},
	}

	stmt, err := ab.db.Prepare(`
		INSERT OR IGNORE INTO ai_training_data (domain, category, confidence, source)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, data := range trainingData {
		stmt.Exec(data.domain, data.category, 1.0, "seed")
	}

	return nil
}

func (ab *AIBlocker) loadTrainingData() error {
	rows, err := ab.db.Query(`
		SELECT domain, category FROM ai_training_data
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var domain, category string
		rows.Scan(&domain, &category)
		ab.model.Train(domain, category)
	}

	return nil
}

// ── Feature Extraction ───────────────────────────────────────────

func (ab *AIBlocker) extractFeatures(domain string) DomainFeatures {
	domain = strings.ToLower(domain)
	
	features := DomainFeatures{
		Length:     len(domain),
		DotCount:   strings.Count(domain, "."),
		HasHyphens: strings.Contains(domain, "-"),
	}

	// Count digits
	for _, ch := range domain {
		if ch >= '0' && ch <= '9' {
			features.DigitCount++
			features.HasNumbers = true
		}
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '.' || ch == '-') {
			features.SpecialCharCount++
		}
	}

	// Extract TLD
	parts := strings.Split(domain, ".")
	if len(parts) > 0 {
		features.TLD = parts[len(parts)-1]
	}

	// Calculate entropy (randomness)
	features.Entropy = ab.calculateEntropy(domain)

	// Check for suspicious words
	suspiciousWords := []string{"free", "porn", "xxx", "sex", "crack", "hack", "casino", "bet"}
	for _, word := range suspiciousWords {
		if strings.Contains(domain, word) {
			features.HasSuspiciousWords = true
			break
		}
	}

	return features
}

func (ab *AIBlocker) calculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[rune]int)
	for _, ch := range s {
		freq[ch]++
	}

	entropy := 0.0
	length := float64(len(s))

	for _, count := range freq {
		p := float64(count) / length
		entropy -= p * math.Log2(p)
	}

	return entropy
}

// ── Naive Bayes Model ────────────────────────────────────────────

func NewNaiveBayesModel() *NaiveBayesModel {
	return &NaiveBayesModel{
		vocabulary:        make(map[string]int),
		categoryCount:     make(map[string]int),
		wordCategoryCount: make(map[string]map[string]int),
		totalDocs:         0,
	}
}

func (m *NaiveBayesModel) Train(domain, category string) {
	m.totalDocs++
	m.categoryCount[category]++

	// Extract words from domain
	words := m.extractWords(domain)
	
	for _, word := range words {
		m.vocabulary[word]++
		
		if _, exists := m.wordCategoryCount[word]; !exists {
			m.wordCategoryCount[word] = make(map[string]int)
		}
		m.wordCategoryCount[word][category]++
	}
}

func (m *NaiveBayesModel) Predict(domain string) (string, float64) {
	words := m.extractWords(domain)
	
	bestCategory := "safe"
	bestScore := -math.MaxFloat64

	for category := range m.categoryCount {
		score := m.calculateScore(words, category)
		if score > bestScore {
			bestScore = score
			bestCategory = category
		}
	}

	// Convert score to probability (0-100%)
	confidence := scoreToConfidence(bestScore)

	return bestCategory, confidence
}

func (m *NaiveBayesModel) calculateScore(words []string, category string) float64 {
	// Prior probability
	prior := math.Log(float64(m.categoryCount[category]) / float64(m.totalDocs))
	
	// Likelihood
	likelihood := 0.0
	vocabularySize := float64(len(m.vocabulary))

	for _, word := range words {
		wordCount := float64(m.wordCategoryCount[word][category])
		categoryTotal := float64(m.categoryCount[category])
		
		// Laplace smoothing
		probability := (wordCount + 1) / (categoryTotal + vocabularySize)
		likelihood += math.Log(probability)
	}

	return prior + likelihood
}

func (m *NaiveBayesModel) extractWords(domain string) []string {
	domain = strings.ToLower(domain)
	
	// Split by dots and hyphens
	domain = strings.ReplaceAll(domain, ".", " ")
	domain = strings.ReplaceAll(domain, "-", " ")
	
	// Also extract character n-grams (3-char substrings)
	var words []string
	parts := strings.Fields(domain)
	
	for _, part := range parts {
		words = append(words, part)
		
		// Add 3-grams
		if len(part) >= 3 {
			for i := 0; i <= len(part)-3; i++ {
				words = append(words, part[i:i+3])
			}
		}
	}

	return words
}

// ── Prediction ───────────────────────────────────────────────────

func (ab *AIBlocker) Predict(domain string) PredictionResult {
	if !ab.enabled {
		return PredictionResult{
			Domain:     domain,
			Category:   "safe",
			Confidence: 0,
			Blocked:    false,
			Reason:     "AI blocker disabled",
		}
	}

	ab.mu.RLock()
	defer ab.mu.RUnlock()

	// Extract features
	features := ab.extractFeatures(domain)

	// Get ML prediction
	category, confidence := ab.model.Predict(domain)

	// Adjust confidence based on features
	adjustedConfidence := ab.adjustConfidence(confidence, features)

	// Determine if should block
	blocked := ab.shouldBlock(category, adjustedConfidence, features)

	result := PredictionResult{
		Domain:     domain,
		Category:   category,
		Confidence: adjustedConfidence,
		Blocked:    blocked,
		Reason:     ab.getReason(category, adjustedConfidence, features),
		Features: map[string]float64{
			"length":         float64(features.Length),
			"digits":         float64(features.DigitCount),
			"entropy":        features.Entropy,
			"has_suspicious": boolToFloat(features.HasSuspiciousWords),
		},
	}

	// Log prediction
	ab.logPrediction(result)

	return result
}

func (ab *AIBlocker) adjustConfidence(baseConfidence float64, features DomainFeatures) float64 {
	confidence := baseConfidence

	// Boost confidence for suspicious TLDs
	suspiciousTLDs := []string{"xxx", "adult", "sex", "porn", "cam"}
	for _, tld := range suspiciousTLDs {
		if features.TLD == tld {
			confidence += 30.0
			break
		}
	}

	// Boost for suspicious words
	if features.HasSuspiciousWords {
		confidence += 15.0
	}

	// Boost for excessive numbers (common in spam/adult sites)
	if float64(features.DigitCount)/float64(features.Length) > 0.3 {
		confidence += 10.0
	}

	// Boost for high entropy (random-looking domains)
	if features.Entropy > 3.5 {
		confidence += 10.0
	}

	// Cap at 100%
	if confidence > 100 {
		confidence = 100
	}

	return confidence
}

func (ab *AIBlocker) shouldBlock(category string, confidence float64, features DomainFeatures) bool {
	// Block thresholds by category
	thresholds := map[string]float64{
		"adult":      60.0, // High sensitivity for adult content
		"gambling":   70.0,
		"malware":    80.0,
		"suspicious": 75.0,
		"safe":       100.0, // Never block safe
	}

	threshold, exists := thresholds[category]
	if !exists {
		threshold = 80.0 // Default threshold
	}

	return confidence >= threshold
}

func (ab *AIBlocker) getReason(category string, confidence float64, features DomainFeatures) string {
	if category == "safe" {
		return "AI classified as safe"
	}

	reasons := []string{
		fmt.Sprintf("AI detected %s content patterns", category),
		fmt.Sprintf("Confidence: %.1f%%", confidence),
	}

	if features.HasSuspiciousWords {
		reasons = append(reasons, "Contains suspicious keywords")
	}

	if features.TLD != "" && ab.isSuspiciousTLD(features.TLD) {
		reasons = append(reasons, fmt.Sprintf("Suspicious TLD: .%s", features.TLD))
	}

	return strings.Join(reasons, ". ")
}

func (ab *AIBlocker) isSuspiciousTLD(tld string) bool {
	suspicious := []string{"xxx", "adult", "sex", "porn", "cam", "bet", "casino"}
	for _, s := range suspicious {
		if tld == s {
			return true
		}
	}
	return false
}

// ── Learning & Improvement ───────────────────────────────────────

func (ab *AIBlocker) LearnFromBlock(domain, category string) error {
	_, err := ab.db.Exec(`
		INSERT OR REPLACE INTO ai_training_data (domain, category, confidence, source)
		VALUES (?, ?, ?, ?)
	`, domain, category, 1.0, "user_block")

	if err != nil {
		return err
	}

	// Retrain model
	ab.mu.Lock()
	defer ab.mu.Unlock()
	
	ab.model.Train(domain, category)

	return nil
}

func (ab *AIBlocker) ReportFalsePositive(domain, predictedCategory, actualCategory, reportedBy string) error {
	_, err := ab.db.Exec(`
		INSERT INTO ai_false_positives (domain, predicted_category, actual_category, reported_by)
		VALUES (?, ?, ?, ?)
	`, domain, predictedCategory, actualCategory, reportedBy)

	if err != nil {
		return err
	}

	// Update training data
	return ab.LearnFromBlock(domain, actualCategory)
}

// ── Statistics ───────────────────────────────────────────────────

func (ab *AIBlocker) logPrediction(result PredictionResult) {
	ab.db.Exec(`
		INSERT INTO ai_predictions (domain, category, confidence, blocked, features)
		VALUES (?, ?, ?, ?, ?)
	`, result.Domain, result.Category, result.Confidence, result.Blocked, 
	   fmt.Sprintf("%v", result.Features))
}

func (ab *AIBlocker) GetModelStats() map[string]interface{} {
	var totalPredictions, falsePositives int
	var accuracy float64

	ab.db.QueryRow(`
		SELECT COUNT(*) FROM ai_predictions
	`).Scan(&totalPredictions)

	ab.db.QueryRow(`
		SELECT COUNT(*) FROM ai_false_positives
	`).Scan(&falsePositives)

	if totalPredictions > 0 {
		accuracy = float64(totalPredictions-falsePositives) / float64(totalPredictions) * 100
	}

	return map[string]interface{}{
		"total_predictions": totalPredictions,
		"false_positives":   falsePositives,
		"accuracy":          accuracy,
		"training_samples":  ab.model.totalDocs,
		"vocabulary_size":   len(ab.model.vocabulary),
		"categories":        len(ab.model.categoryCount),
	}
}

func (ab *AIBlocker) GetRecentPredictions(limit int) ([]PredictionResult, error) {
	rows, err := ab.db.Query(`
		SELECT domain, category, confidence, blocked
		FROM ai_predictions
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []PredictionResult
	for rows.Next() {
		var result PredictionResult
		rows.Scan(&result.Domain, &result.Category, &result.Confidence, &result.Blocked)
		results = append(results, result)
	}

	return results, nil
}

// ── Configuration ────────────────────────────────────────────────

func (ab *AIBlocker) Enable() {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.enabled = true
}

func (ab *AIBlocker) Disable() {
	ab.mu.Lock()
	defer ab.mu.Unlock()
	ab.enabled = false
}

func (ab *AIBlocker) IsEnabled() bool {
	ab.mu.RLock()
	defer ab.mu.RUnlock()
	return ab.enabled
}

// ── Helpers ──────────────────────────────────────────────────────

func scoreToConfidence(score float64) float64 {
	// Convert log probability to percentage (0-100)
	// This is a simplified conversion
	confidence := (1.0 / (1.0 + math.Exp(-score))) * 100
	
	if confidence < 0 {
		confidence = 0
	}
	if confidence > 100 {
		confidence = 100
	}

	return confidence
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

// ── Bulk Training ────────────────────────────────────────────────

func (ab *AIBlocker) TrainBulk(data []struct{ Domain, Category string }) error {
	tx, err := ab.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO ai_training_data (domain, category, confidence, source)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	ab.mu.Lock()
	defer ab.mu.Unlock()

	for _, item := range data {
		stmt.Exec(item.Domain, item.Category, 1.0, "bulk_import")
		ab.model.Train(item.Domain, item.Category)
	}

	return tx.Commit()
}
