package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"stocky/internal/models"
)

type LedgerService struct {
	db *sql.DB
}

func NewLedgerService(db *sql.DB) *LedgerService {
	return &LedgerService{db: db}
}

// GetLedgerEntries retrieves ledger entries with filtering options
func (s *LedgerService) GetLedgerEntries(filter *LedgerFilter) ([]models.LedgerEntry, error) {
	query := `
		SELECT id, event_id, entry_type, account_type, user_id, stock_symbol, 
			   amount, shares, description, timestamp, created_at
		FROM ledger_entries
		WHERE 1=1`
	
	args := []interface{}{}
	argIndex := 1

	if filter.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, filter.UserID)
		argIndex++
	}

	if filter.EventID != "" {
		query += fmt.Sprintf(" AND event_id = $%d", argIndex)
		args = append(args, filter.EventID)
		argIndex++
	}

	if filter.EntryType != "" {
		query += fmt.Sprintf(" AND entry_type = $%d", argIndex)
		args = append(args, filter.EntryType)
		argIndex++
	}

	if filter.AccountType != "" {
		query += fmt.Sprintf(" AND account_type = $%d", argIndex)
		args = append(args, filter.AccountType)
		argIndex++
	}

	if !filter.StartDate.IsZero() {
		query += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, filter.StartDate)
		argIndex++
	}

	if !filter.EndDate.IsZero() {
		query += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, filter.EndDate)
		argIndex++
	}

	query += " ORDER BY timestamp DESC, id DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger entries: %w", err)
	}
	defer rows.Close()

	var entries []models.LedgerEntry
	for rows.Next() {
		var entry models.LedgerEntry
		err := rows.Scan(&entry.ID, &entry.EventID, &entry.EntryType, &entry.AccountType,
			&entry.UserID, &entry.StockSymbol, &entry.Amount, &entry.Shares,
			&entry.Description, &entry.Timestamp, &entry.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// CreateAdjustment creates adjustment entries for correcting previous rewards
func (s *LedgerService) CreateAdjustment(req *AdjustmentRequest) error {
	// Validate the original event exists
	originalEvent, err := s.getRewardEvent(req.OriginalEventID)
	if err != nil {
		return fmt.Errorf("original event not found: %w", err)
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	adjustmentEventID := uuid.New().String()
	timestamp := time.Now()

	// Create adjustment reward event
	adjustmentEvent := &models.RewardEvent{
		EventID:     adjustmentEventID,
		UserID:      originalEvent.UserID,
		StockSymbol: originalEvent.StockSymbol,
		Shares:      req.SharesAdjustment,
		RewardType:  "ADJUSTMENT",
		Timestamp:   timestamp,
	}

	query := `
		INSERT INTO reward_events (event_id, user_id, stock_symbol, shares, reward_type, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)`
	
	_, err = tx.Exec(query, adjustmentEvent.EventID, adjustmentEvent.UserID,
		adjustmentEvent.StockSymbol, adjustmentEvent.Shares, adjustmentEvent.RewardType,
		adjustmentEvent.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create adjustment event: %w", err)
	}

	// Create corresponding ledger entries
	if err := s.createAdjustmentLedgerEntries(tx, adjustmentEvent, req, originalEvent); err != nil {
		return fmt.Errorf("failed to create adjustment ledger entries: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit adjustment: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"adjustment_event_id": adjustmentEventID,
		"original_event_id":   req.OriginalEventID,
		"shares_adjustment":   req.SharesAdjustment,
		"reason":              req.Reason,
	}).Info("Adjustment created successfully")

	return nil
}

// CreateRefund creates refund entries for reversing rewards
func (s *LedgerService) CreateRefund(req *RefundRequest) error {
	// Validate the original event exists
	originalEvent, err := s.getRewardEvent(req.OriginalEventID)
	if err != nil {
		return fmt.Errorf("original event not found: %w", err)
	}

	// Check if already refunded
	if s.isEventRefunded(req.OriginalEventID) {
		return fmt.Errorf("event %s has already been refunded", req.OriginalEventID)
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	refundEventID := uuid.New().String()
	timestamp := time.Now()

	// Create refund reward event (negative shares)
	refundEvent := &models.RewardEvent{
		EventID:     refundEventID,
		UserID:      originalEvent.UserID,
		StockSymbol: originalEvent.StockSymbol,
		Shares:      -originalEvent.Shares, // Negative to reverse
		RewardType:  "REFUND",
		Timestamp:   timestamp,
	}

	query := `
		INSERT INTO reward_events (event_id, user_id, stock_symbol, shares, reward_type, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)`
	
	_, err = tx.Exec(query, refundEvent.EventID, refundEvent.UserID,
		refundEvent.StockSymbol, refundEvent.Shares, refundEvent.RewardType,
		refundEvent.Timestamp)
	if err != nil {
		return fmt.Errorf("failed to create refund event: %w", err)
	}

	// Create corresponding ledger entries (reverse of original)
	if err := s.createRefundLedgerEntries(tx, refundEvent, originalEvent, req.Reason); err != nil {
		return fmt.Errorf("failed to create refund ledger entries: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit refund: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"refund_event_id":   refundEventID,
		"original_event_id": req.OriginalEventID,
		"refunded_shares":   originalEvent.Shares,
		"reason":            req.Reason,
	}).Info("Refund created successfully")

	return nil
}

// GetAccountBalance calculates account balances
func (s *LedgerService) GetAccountBalance(accountType string, userID *string, stockSymbol *string) (*AccountBalance, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN entry_type LIKE '%CREDIT%' THEN COALESCE(amount, 0) ELSE -COALESCE(amount, 0) END), 0) as cash_balance,
			COALESCE(SUM(CASE WHEN entry_type LIKE '%CREDIT%' THEN COALESCE(shares, 0) ELSE -COALESCE(shares, 0) END), 0) as stock_balance,
			COUNT(*) as entry_count
		FROM ledger_entries
		WHERE account_type = $1`
	
	args := []interface{}{accountType}
	argIndex := 2

	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, *userID)
		argIndex++
	}

	if stockSymbol != nil {
		query += fmt.Sprintf(" AND stock_symbol = $%d", argIndex)
		args = append(args, *stockSymbol)
		argIndex++
	}

	var balance AccountBalance
	err := s.db.QueryRow(query, args...).Scan(&balance.CashBalance, &balance.StockBalance, &balance.EntryCount)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate account balance: %w", err)
	}

	balance.AccountType = accountType
	balance.UserID = userID
	balance.StockSymbol = stockSymbol
	balance.LastUpdated = time.Now()

	return &balance, nil
}

// ValidateLedgerIntegrity checks if the ledger is balanced
func (s *LedgerService) ValidateLedgerIntegrity() (*LedgerIntegrityReport, error) {
	report := &LedgerIntegrityReport{
		CheckTime: time.Now(),
		IsValid:   true,
		Issues:    []string{},
	}

	// Check 1: Every event should have balanced entries
	eventBalanceQuery := `
		SELECT event_id, 
			   SUM(CASE WHEN entry_type LIKE '%CREDIT%' THEN COALESCE(amount, 0) ELSE -COALESCE(amount, 0) END) as net_amount,
			   SUM(CASE WHEN entry_type LIKE '%CREDIT%' THEN COALESCE(shares, 0) ELSE -COALESCE(shares, 0) END) as net_shares
		FROM ledger_entries
		GROUP BY event_id
		HAVING ABS(SUM(CASE WHEN entry_type LIKE '%CREDIT%' THEN COALESCE(amount, 0) ELSE -COALESCE(amount, 0) END)) > 0.01`

	rows, err := s.db.Query(eventBalanceQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to check event balance: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var eventID string
		var netAmount, netShares float64
		if err := rows.Scan(&eventID, &netAmount, &netShares); err != nil {
			continue
		}
		report.Issues = append(report.Issues, fmt.Sprintf("Event %s has unbalanced amount: %.4f", eventID, netAmount))
		report.IsValid = false
	}

	// Check 2: User stock balances should match reward events
	userStockQuery := `
		SELECT user_id, stock_symbol,
			   SUM(shares) as reward_shares,
			   (SELECT COALESCE(SUM(CASE WHEN entry_type = 'STOCK_CREDIT' THEN shares ELSE -shares END), 0)
				FROM ledger_entries le 
				WHERE le.user_id = re.user_id AND le.stock_symbol = re.stock_symbol) as ledger_shares
		FROM reward_events re
		GROUP BY user_id, stock_symbol
		HAVING ABS(SUM(shares) - (SELECT COALESCE(SUM(CASE WHEN entry_type = 'STOCK_CREDIT' THEN shares ELSE -shares END), 0)
								  FROM ledger_entries le 
								  WHERE le.user_id = re.user_id AND le.stock_symbol = re.stock_symbol)) > 0.000001`

	rows2, err := s.db.Query(userStockQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to check user stock balance: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var userID, stockSymbol string
		var rewardShares, ledgerShares float64
		if err := rows2.Scan(&userID, &stockSymbol, &rewardShares, &ledgerShares); err != nil {
			continue
		}
		report.Issues = append(report.Issues, 
			fmt.Sprintf("User %s stock %s: reward_shares=%.6f, ledger_shares=%.6f", 
				userID, stockSymbol, rewardShares, ledgerShares))
		report.IsValid = false
	}

	// Check 3: Orphaned ledger entries
	orphanQuery := `
		SELECT COUNT(*) FROM ledger_entries le
		LEFT JOIN reward_events re ON le.event_id = re.event_id
		WHERE re.event_id IS NULL`

	var orphanCount int
	if err := s.db.QueryRow(orphanQuery).Scan(&orphanCount); err == nil && orphanCount > 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("Found %d orphaned ledger entries", orphanCount))
		report.IsValid = false
	}

	return report, nil
}

// Helper functions

func (s *LedgerService) getRewardEvent(eventID string) (*models.RewardEvent, error) {
	query := `
		SELECT id, event_id, user_id, stock_symbol, shares, reward_type, timestamp, created_at
		FROM reward_events
		WHERE event_id = $1`
	
	var event models.RewardEvent
	err := s.db.QueryRow(query, eventID).Scan(&event.ID, &event.EventID, &event.UserID,
		&event.StockSymbol, &event.Shares, &event.RewardType, &event.Timestamp, &event.CreatedAt)
	
	if err != nil {
		return nil, err
	}
	
	return &event, nil
}

func (s *LedgerService) isEventRefunded(eventID string) bool {
	query := `SELECT COUNT(*) FROM reward_events WHERE reward_type = 'REFUND' AND event_id LIKE $1`
	var count int
	s.db.QueryRow(query, "%"+eventID+"%").Scan(&count)
	return count > 0
}

func (s *LedgerService) createAdjustmentLedgerEntries(tx *sql.Tx, adjustmentEvent *models.RewardEvent, req *AdjustmentRequest, originalEvent *models.RewardEvent) error {
	// Calculate adjustment amounts
	currentPrice := 100.0 // Simplified - should get from price service
	adjustmentValue := req.SharesAdjustment * currentPrice
	
	entries := []models.LedgerEntry{
		{
			EventID:     adjustmentEvent.EventID,
			EntryType:   "STOCK_CREDIT",
			AccountType: "USER_STOCK",
			UserID:      &adjustmentEvent.UserID,
			StockSymbol: &adjustmentEvent.StockSymbol,
			Shares:      &req.SharesAdjustment,
			Description: fmt.Sprintf("Adjustment: %s (Original Event: %s)", req.Reason, req.OriginalEventID),
			Timestamp:   adjustmentEvent.Timestamp,
		},
		{
			EventID:     adjustmentEvent.EventID,
			EntryType:   "CASH_DEBIT",
			AccountType: "COMPANY_CASH",
			Amount:      &adjustmentValue,
			Description: fmt.Sprintf("Adjustment cost: %s", req.Reason),
			Timestamp:   adjustmentEvent.Timestamp,
		},
	}

	return s.insertLedgerEntries(tx, entries)
}

func (s *LedgerService) createRefundLedgerEntries(tx *sql.Tx, refundEvent *models.RewardEvent, originalEvent *models.RewardEvent, reason string) error {
	// Create reverse entries
	currentPrice := 100.0 // Simplified
	refundValue := originalEvent.Shares * currentPrice
	
	entries := []models.LedgerEntry{
		{
			EventID:     refundEvent.EventID,
			EntryType:   "STOCK_DEBIT",
			AccountType: "USER_STOCK",
			UserID:      &refundEvent.UserID,
			StockSymbol: &refundEvent.StockSymbol,
			Shares:      &originalEvent.Shares, // Positive value for debit
			Description: fmt.Sprintf("Refund: %s (Original Event: %s)", reason, originalEvent.EventID),
			Timestamp:   refundEvent.Timestamp,
		},
		{
			EventID:     refundEvent.EventID,
			EntryType:   "CASH_CREDIT",
			AccountType: "COMPANY_CASH",
			Amount:      &refundValue,
			Description: fmt.Sprintf("Refund recovery: %s", reason),
			Timestamp:   refundEvent.Timestamp,
		},
	}

	return s.insertLedgerEntries(tx, entries)
}

func (s *LedgerService) insertLedgerEntries(tx *sql.Tx, entries []models.LedgerEntry) error {
	for _, entry := range entries {
		query := `
			INSERT INTO ledger_entries (event_id, entry_type, account_type, user_id, stock_symbol, amount, shares, description, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		
		_, err := tx.Exec(query, entry.EventID, entry.EntryType, entry.AccountType,
			entry.UserID, entry.StockSymbol, entry.Amount, entry.Shares, entry.Description, entry.Timestamp)
		
		if err != nil {
			return fmt.Errorf("failed to insert ledger entry: %w", err)
		}
	}
	return nil
}

// Data structures for ledger operations

type LedgerFilter struct {
	UserID      string    `json:"user_id"`
	EventID     string    `json:"event_id"`
	EntryType   string    `json:"entry_type"`
	AccountType string    `json:"account_type"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Limit       int       `json:"limit"`
}

type AdjustmentRequest struct {
	OriginalEventID   string  `json:"original_event_id" binding:"required"`
	SharesAdjustment  float64 `json:"shares_adjustment" binding:"required"`
	Reason            string  `json:"reason" binding:"required"`
}

type RefundRequest struct {
	OriginalEventID string `json:"original_event_id" binding:"required"`
	Reason          string `json:"reason" binding:"required"`
}

type AccountBalance struct {
	AccountType  string    `json:"account_type"`
	UserID       *string   `json:"user_id,omitempty"`
	StockSymbol  *string   `json:"stock_symbol,omitempty"`
	CashBalance  float64   `json:"cash_balance"`
	StockBalance float64   `json:"stock_balance"`
	EntryCount   int       `json:"entry_count"`
	LastUpdated  time.Time `json:"last_updated"`
}

type LedgerIntegrityReport struct {
	CheckTime time.Time `json:"check_time"`
	IsValid   bool      `json:"is_valid"`
	Issues    []string  `json:"issues"`
}
