package models

import (
	"time"
)

type RewardEvent struct {
	ID          int       `json:"id" db:"id"`
	EventID     string    `json:"event_id" db:"event_id"`
	UserID      string    `json:"user_id" db:"user_id"`
	StockSymbol string    `json:"stock_symbol" db:"stock_symbol"`
	Shares      float64   `json:"shares" db:"shares"`
	RewardType  string    `json:"reward_type" db:"reward_type"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type LedgerEntry struct {
	ID          int       `json:"id" db:"id"`
	EventID     string    `json:"event_id" db:"event_id"`
	EntryType   string    `json:"entry_type" db:"entry_type"`
	AccountType string    `json:"account_type" db:"account_type"`
	UserID      *string   `json:"user_id,omitempty" db:"user_id"`
	StockSymbol *string   `json:"stock_symbol,omitempty" db:"stock_symbol"`
	Amount      *float64  `json:"amount,omitempty" db:"amount"`
	Shares      *float64  `json:"shares,omitempty" db:"shares"`
	Description string    `json:"description" db:"description"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type StockPrice struct {
	ID          int       `json:"id" db:"id"`
	StockSymbol string    `json:"stock_symbol" db:"stock_symbol"`
	Price       float64   `json:"price" db:"price"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Stock struct {
	ID       int    `json:"id" db:"id"`
	Symbol   string `json:"symbol" db:"symbol"`
	Name     string `json:"name" db:"name"`
	Exchange string `json:"exchange" db:"exchange"`
}

type RewardRequest struct {
	UserID      string    `json:"user_id" binding:"required"`
	StockSymbol string    `json:"stock_symbol" binding:"required"`
	Shares      float64   `json:"shares" binding:"required,gt=0"`
	RewardType  string    `json:"reward_type" binding:"required"`
	Timestamp   time.Time `json:"timestamp"`
}

type TodayStocksResponse struct {
	UserID string         `json:"user_id"`
	Date   string         `json:"date"`
	Stocks []RewardEvent  `json:"stocks"`
}

type HistoricalINRResponse struct {
	UserID string                    `json:"user_id"`
	Days   []HistoricalINRDayData    `json:"days"`
}

type HistoricalINRDayData struct {
	Date      string  `json:"date"`
	INRValue  float64 `json:"inr_value"`
}

type StatsResponse struct {
	UserID              string                 `json:"user_id"`
	TotalSharesToday    []StockSharesSummary   `json:"total_shares_today"`
	CurrentINRValue     float64                `json:"current_inr_value"`
}

type StockSharesSummary struct {
	StockSymbol string  `json:"stock_symbol"`
	TotalShares float64 `json:"total_shares"`
}

type PortfolioResponse struct {
	UserID   string            `json:"user_id"`
	Holdings []PortfolioHolding `json:"holdings"`
	TotalINRValue float64       `json:"total_inr_value"`
}

type PortfolioHolding struct {
	StockSymbol   string  `json:"stock_symbol"`
	TotalShares   float64 `json:"total_shares"`
	CurrentPrice  float64 `json:"current_price"`
	INRValue      float64 `json:"inr_value"`
}
