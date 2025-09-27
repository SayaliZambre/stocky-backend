package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"stocky/internal/models"
)

type RewardService struct {
	db *sql.DB
}

func NewRewardService(db *sql.DB) *RewardService {
	return &RewardService{db: db}
}

func (s *RewardService) CreateReward(req *models.RewardRequest) (*models.RewardEvent, error) {
	// Generate unique event ID to prevent duplicates
	eventID := uuid.New().String()
	
	// Use provided timestamp or current time
	timestamp := req.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Ensure user exists
	if err := s.ensureUserExists(tx, req.UserID); err != nil {
		return nil, fmt.Errorf("failed to ensure user exists: %w", err)
	}

	// Ensure stock exists
	if err := s.ensureStockExists(tx, req.StockSymbol); err != nil {
		return nil, fmt.Errorf("failed to ensure stock exists: %w", err)
	}

	// Create reward event
	rewardEvent := &models.RewardEvent{
		EventID:     eventID,
		UserID:      req.UserID,
		StockSymbol: req.StockSymbol,
		Shares:      req.Shares,
		RewardType:  req.RewardType,
		Timestamp:   timestamp,
	}

	query := `
		INSERT INTO reward_events (event_id, user_id, stock_symbol, shares, reward_type, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	
	err = tx.QueryRow(query, rewardEvent.EventID, rewardEvent.UserID, rewardEvent.StockSymbol,
		rewardEvent.Shares, rewardEvent.RewardType, rewardEvent.Timestamp).
		Scan(&rewardEvent.ID, &rewardEvent.CreatedAt)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create reward event: %w", err)
	}

	// Create ledger entries for double-entry accounting
	if err := s.createLedgerEntries(tx, rewardEvent); err != nil {
		return nil, fmt.Errorf("failed to create ledger entries: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"event_id": eventID,
		"user_id": req.UserID,
		"stock_symbol": req.StockSymbol,
		"shares": req.Shares,
	}).Info("Reward created successfully")

	return rewardEvent, nil
}

func (s *RewardService) GetTodayStocks(userID string) (*models.TodayStocksResponse, error) {
	today := time.Now().Format("2006-01-02")
	
	query := `
		SELECT id, event_id, user_id, stock_symbol, shares, reward_type, timestamp, created_at
		FROM reward_events
		WHERE user_id = $1 AND DATE(timestamp) = $2
		ORDER BY timestamp DESC`
	
	rows, err := s.db.Query(query, userID, today)
	if err != nil {
		return nil, fmt.Errorf("failed to query today's stocks: %w", err)
	}
	defer rows.Close()

	var stocks []models.RewardEvent
	for rows.Next() {
		var stock models.RewardEvent
		err := rows.Scan(&stock.ID, &stock.EventID, &stock.UserID, &stock.StockSymbol,
			&stock.Shares, &stock.RewardType, &stock.Timestamp, &stock.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan stock row: %w", err)
		}
		stocks = append(stocks, stock)
	}

	return &models.TodayStocksResponse{
		UserID: userID,
		Date:   today,
		Stocks: stocks,
	}, nil
}

func (s *RewardService) ensureUserExists(tx *sql.Tx, userID string) error {
	query := `INSERT INTO users (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`
	_, err := tx.Exec(query, userID)
	return err
}

func (s *RewardService) ensureStockExists(tx *sql.Tx, symbol string) error {
	// Check if stock exists
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM stocks WHERE symbol = $1)`
	err := tx.QueryRow(query, symbol).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("stock symbol %s does not exist", symbol)
	}

	return nil
}

func (s *RewardService) createLedgerEntries(tx *sql.Tx, event *models.RewardEvent) error {
	// Get current stock price for calculations
	currentPrice, err := s.getCurrentStockPrice(tx, event.StockSymbol)
	if err != nil {
		logrus.Warnf("Could not get current price for %s, using default: %v", event.StockSymbol, err)
		currentPrice = 100.0 // Default price for calculation
	}

	stockValue := event.Shares * currentPrice
	brokerageFee := stockValue * 0.001  // 0.1% brokerage
	sttFee := stockValue * 0.001        // 0.1% STT
	gstFee := brokerageFee * 0.18       // 18% GST on brokerage
	totalFees := brokerageFee + sttFee + gstFee
	totalCost := stockValue + totalFees

	entries := []models.LedgerEntry{
		// Credit stock to user
		{
			EventID:     event.EventID,
			EntryType:   "STOCK_CREDIT",
			AccountType: "USER_STOCK",
			UserID:      &event.UserID,
			StockSymbol: &event.StockSymbol,
			Shares:      &event.Shares,
			Description: fmt.Sprintf("Stock reward: %s shares of %s", fmt.Sprintf("%.6f", event.Shares), event.StockSymbol),
			Timestamp:   event.Timestamp,
		},
		// Debit cash from company for stock purchase
		{
			EventID:     event.EventID,
			EntryType:   "CASH_DEBIT",
			AccountType: "COMPANY_CASH",
			Amount:      &stockValue,
			Description: fmt.Sprintf("Stock purchase cost for %s shares of %s", fmt.Sprintf("%.6f", event.Shares), event.StockSymbol),
			Timestamp:   event.Timestamp,
		},
		// Debit cash from company for fees
		{
			EventID:     event.EventID,
			EntryType:   "FEE_DEBIT",
			AccountType: "COMPANY_FEES",
			Amount:      &totalFees,
			Description: fmt.Sprintf("Trading fees (brokerage: %.4f, STT: %.4f, GST: %.4f)", brokerageFee, sttFee, gstFee),
			Timestamp:   event.Timestamp,
		},
	}

	for _, entry := range entries {
		query := `
			INSERT INTO ledger_entries (event_id, entry_type, account_type, user_id, stock_symbol, amount, shares, description, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
		
		_, err := tx.Exec(query, entry.EventID, entry.EntryType, entry.AccountType,
			entry.UserID, entry.StockSymbol, entry.Amount, entry.Shares, entry.Description, entry.Timestamp)
		
		if err != nil {
			return fmt.Errorf("failed to create ledger entry: %w", err)
		}
	}

	return nil
}

func (s *RewardService) getCurrentStockPrice(tx *sql.Tx, symbol string) (float64, error) {
	var price float64
	query := `
		SELECT price FROM stock_prices 
		WHERE stock_symbol = $1 
		ORDER BY timestamp DESC 
		LIMIT 1`
	
	err := tx.QueryRow(query, symbol).Scan(&price)
	if err != nil {
		return 0, err
	}
	
	return price, nil
}
