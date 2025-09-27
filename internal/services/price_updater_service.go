package services

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"stocky/internal/models"
)

type PriceUpdaterService struct {
	db           *sql.DB
	stockService *StockService
	ticker       *time.Ticker
	stopChan     chan bool
	mu           sync.RWMutex
	lastUpdate   time.Time
	isRunning    bool
}

func NewPriceUpdaterService(db *sql.DB, stockService *StockService) *PriceUpdaterService {
	return &PriceUpdaterService{
		db:           db,
		stockService: stockService,
		stopChan:     make(chan bool),
	}
}

// StartPriceUpdater starts the hourly price update service
func (s *PriceUpdaterService) StartPriceUpdater() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		logrus.Warn("Price updater is already running")
		return
	}

	s.ticker = time.NewTicker(1 * time.Hour)
	s.isRunning = true

	logrus.Info("Starting price updater service")

	// Run initial update
	go func() {
		if err := s.updateAllStockPrices(); err != nil {
			logrus.WithError(err).Error("Initial price update failed")
		}
	}()

	// Start periodic updates
	go func() {
		for {
			select {
			case <-s.ticker.C:
				if err := s.updateAllStockPrices(); err != nil {
					logrus.WithError(err).Error("Scheduled price update failed")
				}
			case <-s.stopChan:
				s.ticker.Stop()
				logrus.Info("Price updater service stopped")
				return
			}
		}
	}()
}

// StopPriceUpdater stops the price update service
func (s *PriceUpdaterService) StopPriceUpdater() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	s.stopChan <- true
	s.isRunning = false
	logrus.Info("Stopping price updater service")
}

// updateAllStockPrices fetches and updates prices for all stocks
func (s *PriceUpdaterService) updateAllStockPrices() error {
	startTime := time.Now()
	logrus.Info("Starting stock price update")

	// Get all active stock symbols
	symbols, err := s.getActiveStockSymbols()
	if err != nil {
		return fmt.Errorf("failed to get active stock symbols: %w", err)
	}

	if len(symbols) == 0 {
		logrus.Info("No active stocks found for price update")
		return nil
	}

	// Fetch prices with retry logic
	prices, err := s.fetchPricesWithRetry(symbols, 3)
	if err != nil {
		return fmt.Errorf("failed to fetch prices after retries: %w", err)
	}

	// Update prices in database
	updatedCount, err := s.updatePricesInDB(prices)
	if err != nil {
		return fmt.Errorf("failed to update prices in database: %w", err)
	}

	s.mu.Lock()
	s.lastUpdate = time.Now()
	s.mu.Unlock()

	duration := time.Since(startTime)
	logrus.WithFields(logrus.Fields{
		"symbols_requested": len(symbols),
		"prices_updated":    updatedCount,
		"duration":          duration,
	}).Info("Stock price update completed")

	return nil
}

// getActiveStockSymbols returns all stock symbols that have been used in rewards
func (s *PriceUpdaterService) getActiveStockSymbols() ([]string, error) {
	query := `
		SELECT DISTINCT s.symbol
		FROM stocks s
		INNER JOIN reward_events re ON s.symbol = re.stock_symbol
		ORDER BY s.symbol`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, err
		}
		symbols = append(symbols, symbol)
	}

	return symbols, nil
}

// fetchPricesWithRetry attempts to fetch prices with retry logic
func (s *PriceUpdaterService) fetchPricesWithRetry(symbols []string, maxRetries int) (map[string]float64, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logrus.WithFields(logrus.Fields{
			"attempt":     attempt,
			"max_retries": maxRetries,
			"symbols":     len(symbols),
		}).Info("Attempting to fetch stock prices")

		prices, err := s.stockService.GetMultiplePrices(symbols)
		if err != nil {
			lastErr = err
			logrus.WithError(err).Warnf("Price fetch attempt %d failed", attempt)
			
			if attempt < maxRetries {
				// Exponential backoff
				backoff := time.Duration(attempt*attempt) * time.Second
				logrus.Infof("Retrying in %v", backoff)
				time.Sleep(backoff)
			}
			continue
		}

		// Check if we got prices for most symbols (allow some failures)
		successRate := float64(len(prices)) / float64(len(symbols))
		if successRate < 0.5 {
			lastErr = fmt.Errorf("low success rate: got %d prices out of %d symbols", len(prices), len(symbols))
			logrus.WithError(lastErr).Warnf("Price fetch attempt %d had low success rate", attempt)
			continue
		}

		logrus.WithFields(logrus.Fields{
			"symbols_requested": len(symbols),
			"prices_received":   len(prices),
			"success_rate":      fmt.Sprintf("%.1f%%", successRate*100),
		}).Info("Price fetch successful")

		return prices, nil
	}

	return nil, fmt.Errorf("failed to fetch prices after %d attempts: %w", maxRetries, lastErr)
}

// updatePricesInDB updates stock prices in the database
func (s *PriceUpdaterService) updatePricesInDB(prices map[string]float64) (int, error) {
	if len(prices) == 0 {
		return 0, nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	timestamp := time.Now()
	updatedCount := 0

	for symbol, price := range prices {
		// Insert new price record
		query := `
			INSERT INTO stock_prices (stock_symbol, price, timestamp)
			VALUES ($1, $2, $3)
			ON CONFLICT (stock_symbol, timestamp) DO UPDATE SET
				price = EXCLUDED.price,
				created_at = CURRENT_TIMESTAMP`

		_, err := tx.Exec(query, symbol, price, timestamp)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to update price for %s", symbol)
			continue
		}

		updatedCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit price updates: %w", err)
	}

	// Clean up old price records (keep last 30 days)
	go s.cleanupOldPrices()

	return updatedCount, nil
}

// cleanupOldPrices removes price records older than 30 days
func (s *PriceUpdaterService) cleanupOldPrices() {
	query := `
		DELETE FROM stock_prices
		WHERE created_at < NOW() - INTERVAL '30 days'`

	result, err := s.db.Exec(query)
	if err != nil {
		logrus.WithError(err).Error("Failed to cleanup old prices")
		return
	}

	if rowsAffected, err := result.RowsAffected(); err == nil && rowsAffected > 0 {
		logrus.WithField("rows_deleted", rowsAffected).Info("Cleaned up old price records")
	}
}

// GetLastUpdateTime returns the last successful update time
func (s *PriceUpdaterService) GetLastUpdateTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdate
}

// IsRunning returns whether the price updater is currently running
func (s *PriceUpdaterService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// ForceUpdate triggers an immediate price update
func (s *PriceUpdaterService) ForceUpdate() error {
	logrus.Info("Force updating stock prices")
	return s.updateAllStockPrices()
}

// GetStaleDataStatus checks for stale price data
func (s *PriceUpdaterService) GetStaleDataStatus() (*StaleDataStatus, error) {
	query := `
		SELECT 
			stock_symbol,
			MAX(timestamp) as last_update,
			COUNT(*) as price_count
		FROM stock_prices
		GROUP BY stock_symbol
		ORDER BY last_update DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query price status: %w", err)
	}
	defer rows.Close()

	var staleSymbols []string
	var totalSymbols int
	staleThreshold := time.Now().Add(-2 * time.Hour) // Consider stale if older than 2 hours

	for rows.Next() {
		var symbol string
		var lastUpdate time.Time
		var priceCount int

		if err := rows.Scan(&symbol, &lastUpdate, &priceCount); err != nil {
			continue
		}

		totalSymbols++
		if lastUpdate.Before(staleThreshold) {
			staleSymbols = append(staleSymbols, symbol)
		}
	}

	return &StaleDataStatus{
		TotalSymbols:  totalSymbols,
		StaleSymbols:  staleSymbols,
		StaleCount:    len(staleSymbols),
		LastCheck:     time.Now(),
		StaleThreshold: staleThreshold,
	}, nil
}

type StaleDataStatus struct {
	TotalSymbols   int       `json:"total_symbols"`
	StaleSymbols   []string  `json:"stale_symbols"`
	StaleCount     int       `json:"stale_count"`
	LastCheck      time.Time `json:"last_check"`
	StaleThreshold time.Time `json:"stale_threshold"`
}
