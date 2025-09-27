package database

import (
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"
)

func Migrate(db *sql.DB) error {
	migrations := []string{
		createUsersTable,
		createStocksTable,
		createRewardEventsTable,
		createLedgerEntriesTable,
		createStockPricesTable,
		createIndexes,
	}

	for i, migration := range migrations {
		logrus.Infof("Running migration %d", i+1)
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	logrus.Info("All migrations completed successfully")
	return nil
}

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`

const createStocksTable = `
CREATE TABLE IF NOT EXISTS stocks (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    exchange VARCHAR(10) NOT NULL DEFAULT 'NSE',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`

const createRewardEventsTable = `
CREATE TABLE IF NOT EXISTS reward_events (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(255) UNIQUE NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    stock_symbol VARCHAR(50) NOT NULL,
    shares NUMERIC(18, 6) NOT NULL,
    reward_type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (stock_symbol) REFERENCES stocks(symbol)
);`

const createLedgerEntriesTable = `
CREATE TABLE IF NOT EXISTS ledger_entries (
    id SERIAL PRIMARY KEY,
    event_id VARCHAR(255) NOT NULL,
    entry_type VARCHAR(50) NOT NULL, -- 'STOCK_CREDIT', 'CASH_DEBIT', 'FEE_DEBIT'
    account_type VARCHAR(50) NOT NULL, -- 'USER_STOCK', 'COMPANY_CASH', 'COMPANY_FEES'
    user_id VARCHAR(255),
    stock_symbol VARCHAR(50),
    amount NUMERIC(18, 4), -- For INR amounts
    shares NUMERIC(18, 6), -- For stock quantities
    description TEXT,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (event_id) REFERENCES reward_events(event_id)
);`

const createStockPricesTable = `
CREATE TABLE IF NOT EXISTS stock_prices (
    id SERIAL PRIMARY KEY,
    stock_symbol VARCHAR(50) NOT NULL,
    price NUMERIC(18, 4) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (stock_symbol) REFERENCES stocks(symbol),
    UNIQUE(stock_symbol, timestamp)
);`

const createIndexes = `
CREATE INDEX IF NOT EXISTS idx_reward_events_user_id ON reward_events(user_id);
CREATE INDEX IF NOT EXISTS idx_reward_events_timestamp ON reward_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_reward_events_user_date ON reward_events(user_id, DATE(timestamp));
CREATE INDEX IF NOT EXISTS idx_ledger_entries_event_id ON ledger_entries(event_id);
CREATE INDEX IF NOT EXISTS idx_ledger_entries_user_id ON ledger_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_stock_prices_symbol_timestamp ON stock_prices(stock_symbol, timestamp DESC);`
