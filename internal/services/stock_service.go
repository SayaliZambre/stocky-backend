package services

import (
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
)

type StockService struct {
	// In a real implementation, this would connect to NSE/BSE APIs
}

func NewStockService() *StockService {
	return &StockService{}
}

// GetCurrentPrice simulates fetching current stock price
// In production, this would call actual NSE/BSE APIs
func (s *StockService) GetCurrentPrice(symbol string) (float64, error) {
	// Simulate API call delay
	time.Sleep(10 * time.Millisecond)
	
	// Generate realistic stock prices based on symbol
	basePrice := s.getBasePriceForSymbol(symbol)
	
	// Add random variation (-5% to +5%)
	variation := (rand.Float64() - 0.5) * 0.1
	price := basePrice * (1 + variation)
	
	logrus.WithFields(logrus.Fields{
		"symbol": symbol,
		"price":  price,
	}).Debug("Generated stock price")
	
	return price, nil
}

func (s *StockService) getBasePriceForSymbol(symbol string) float64 {
	// Realistic base prices for Indian stocks (in INR)
	basePrices := map[string]float64{
		"RELIANCE":    2500.0,
		"TCS":         3800.0,
		"INFY":        1600.0,
		"HDFCBANK":    1700.0,
		"ICICIBANK":   950.0,
		"HINDUNILVR":  2400.0,
		"ITC":         450.0,
		"SBIN":        600.0,
		"BHARTIARTL":  900.0,
		"KOTAKBANK":   1800.0,
	}
	
	if price, exists := basePrices[symbol]; exists {
		return price
	}
	
	// Default price for unknown stocks
	return 1000.0
}

// GetMultiplePrices fetches prices for multiple stocks
func (s *StockService) GetMultiplePrices(symbols []string) (map[string]float64, error) {
	prices := make(map[string]float64)
	
	for _, symbol := range symbols {
		price, err := s.GetCurrentPrice(symbol)
		if err != nil {
			logrus.WithError(err).Warnf("Failed to get price for %s", symbol)
			continue
		}
		prices[symbol] = price
	}
	
	return prices, nil
}
