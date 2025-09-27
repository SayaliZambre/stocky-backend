package services

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"stocky/internal/models"
)

type PortfolioService struct {
	db           *sql.DB
	stockService *StockService
}

func NewPortfolioService(db *sql.DB, stockService *StockService) *PortfolioService {
	return &PortfolioService{
		db:           db,
		stockService: stockService,
	}
}

func (s *PortfolioService) GetHistoricalINR(userID string) (*models.HistoricalINRResponse, error) {
	// Get all historical days (excluding today)
	query := `
		SELECT DISTINCT DATE(timestamp) as date
		FROM reward_events
		WHERE user_id = $1 AND DATE(timestamp) < CURRENT_DATE
		ORDER BY date DESC`
	
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query historical dates: %w", err)
	}
	defer rows.Close()

	var days []models.HistoricalINRDayData
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			return nil, fmt.Errorf("failed to scan date: %w", err)
		}

		// Calculate INR value for this specific date
		inrValue, err := s.calculateINRValueForDate(userID, dateStr)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to calculate INR value for date %s", dateStr)
			continue
		}

		days = append(days, models.HistoricalINRDayData{
			Date:     dateStr,
			INRValue: inrValue,
		})
	}

	return &models.HistoricalINRResponse{
		UserID: userID,
		Days:   days,
	}, nil
}

func (s *PortfolioService) GetStats(userID string) (*models.StatsResponse, error) {
	// Get today's shares grouped by stock symbol
	today := time.Now().Format("2006-01-02")
	
	query := `
		SELECT stock_symbol, SUM(shares) as total_shares
		FROM reward_events
		WHERE user_id = $1 AND DATE(timestamp) = $2
		GROUP BY stock_symbol
		ORDER BY stock_symbol`
	
	rows, err := s.db.Query(query, userID, today)
	if err != nil {
		return nil, fmt.Errorf("failed to query today's shares: %w", err)
	}
	defer rows.Close()

	var todayShares []models.StockSharesSummary
	for rows.Next() {
		var summary models.StockSharesSummary
		if err := rows.Scan(&summary.StockSymbol, &summary.TotalShares); err != nil {
			return nil, fmt.Errorf("failed to scan shares summary: %w", err)
		}
		todayShares = append(todayShares, summary)
	}

	// Calculate current INR value of entire portfolio
	currentINRValue, err := s.calculateCurrentPortfolioValue(userID)
	if err != nil {
		logrus.WithError(err).Warn("Failed to calculate current portfolio value")
		currentINRValue = 0
	}

	return &models.StatsResponse{
		UserID:              userID,
		TotalSharesToday:    todayShares,
		CurrentINRValue:     currentINRValue,
	}, nil
}

func (s *PortfolioService) GetPortfolio(userID string) (*models.PortfolioResponse, error) {
	// Get total shares by stock symbol for the user
	query := `
		SELECT stock_symbol, SUM(shares) as total_shares
		FROM reward_events
		WHERE user_id = $1
		GROUP BY stock_symbol
		HAVING SUM(shares) > 0
		ORDER BY stock_symbol`
	
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query portfolio holdings: %w", err)
	}
	defer rows.Close()

	var holdings []models.PortfolioHolding
	var symbols []string
	
	// Collect holdings and symbols
	for rows.Next() {
		var holding models.PortfolioHolding
		if err := rows.Scan(&holding.StockSymbol, &holding.TotalShares); err != nil {
			return nil, fmt.Errorf("failed to scan holding: %w", err)
		}
		holdings = append(holdings, holding)
		symbols = append(symbols, holding.StockSymbol)
	}

	// Get current prices for all symbols
	prices, err := s.stockService.GetMultiplePrices(symbols)
	if err != nil {
		return nil, fmt.Errorf("failed to get current prices: %w", err)
	}

	// Calculate INR values
	var totalINRValue float64
	for i := range holdings {
		symbol := holdings[i].StockSymbol
		if price, exists := prices[symbol]; exists {
			holdings[i].CurrentPrice = price
			holdings[i].INRValue = holdings[i].TotalShares * price
			totalINRValue += holdings[i].INRValue
		} else {
			logrus.Warnf("No price found for symbol %s", symbol)
		}
	}

	return &models.PortfolioResponse{
		UserID:        userID,
		Holdings:      holdings,
		TotalINRValue: totalINRValue,
	}, nil
}

func (s *PortfolioService) calculateINRValueForDate(userID, dateStr string) (float64, error) {
	// Get all stocks held by user up to this date
	query := `
		SELECT stock_symbol, SUM(shares) as total_shares
		FROM reward_events
		WHERE user_id = $1 AND DATE(timestamp) <= $2
		GROUP BY stock_symbol
		HAVING SUM(shares) > 0`
	
	rows, err := s.db.Query(query, userID, dateStr)
	if err != nil {
		return 0, fmt.Errorf("failed to query holdings for date: %w", err)
	}
	defer rows.Close()

	var totalValue float64
	for rows.Next() {
		var symbol string
		var shares float64
		if err := rows.Scan(&symbol, &shares); err != nil {
			return 0, fmt.Errorf("failed to scan holding: %w", err)
		}

		// Get historical price for this date (or use current price as approximation)
		price, err := s.getHistoricalPrice(symbol, dateStr)
		if err != nil {
			// Fallback to current price
			price, _ = s.stockService.GetCurrentPrice(symbol)
		}

		totalValue += shares * price
	}

	return totalValue, nil
}

func (s *PortfolioService) calculateCurrentPortfolioValue(userID string) (float64, error) {
	// Get all current holdings
	query := `
		SELECT stock_symbol, SUM(shares) as total_shares
		FROM reward_events
		WHERE user_id = $1
		GROUP BY stock_symbol
		HAVING SUM(shares) > 0`
	
	rows, err := s.db.Query(query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to query current holdings: %w", err)
	}
	defer rows.Close()

	var symbols []string
	var holdings map[string]float64 = make(map[string]float64)
	
	for rows.Next() {
		var symbol string
		var shares float64
		if err := rows.Scan(&symbol, &shares); err != nil {
			return 0, fmt.Errorf("failed to scan holding: %w", err)
		}
		symbols = append(symbols, symbol)
		holdings[symbol] = shares
	}

	// Get current prices
	prices, err := s.stockService.GetMultiplePrices(symbols)
	if err != nil {
		return 0, fmt.Errorf("failed to get current prices: %w", err)
	}

	// Calculate total value
	var totalValue float64
	for symbol, shares := range holdings {
		if price, exists := prices[symbol]; exists {
			totalValue += shares * price
		}
	}

	return totalValue, nil
}

func (s *PortfolioService) getHistoricalPrice(symbol, dateStr string) (float64, error) {
	// Try to get historical price from database
	query := `
		SELECT price FROM stock_prices
		WHERE stock_symbol = $1 AND DATE(timestamp) = $2
		ORDER BY timestamp DESC
		LIMIT 1`
	
	var price float64
	err := s.db.QueryRow(query, symbol, dateStr).Scan(&price)
	if err != nil {
		return 0, fmt.Errorf("no historical price found for %s on %s", symbol, dateStr)
	}
	
	return price, nil
}
