# Stocky - Stock Rewards System

A comprehensive backend system for managing stock rewards where users earn shares of Indian stocks as incentives. Built with Go, Gin, and PostgreSQL with a robust double-entry ledger system.

## Features

- **Stock Reward Management**: Award users shares of Indian stocks (RELIANCE, TCS, INFY, etc.)
- **Double-Entry Ledger**: Complete financial tracking with proper accounting principles
- **Real-time Price Updates**: Hourly stock price fetching with fallback mechanisms
- **Portfolio Management**: Track user holdings and calculate INR valuations
- **Advanced Operations**: Support for adjustments, refunds, and integrity validation
- **Comprehensive APIs**: RESTful endpoints for all operations

## Architecture

### Database Schema

The system uses PostgreSQL with the following key tables:

- **users**: User management
- **stocks**: Stock symbol definitions
- **reward_events**: All reward transactions
- **ledger_entries**: Double-entry accounting records
- **stock_prices**: Historical price data

### Services

- **RewardService**: Handles stock reward creation and retrieval
- **PortfolioService**: Manages user portfolios and statistics
- **StockService**: Simulates stock price fetching (NSE/BSE integration ready)
- **PriceUpdaterService**: Hourly price updates with retry logic
- **LedgerService**: Double-entry accounting and financial operations

## API Endpoints

### Core Reward APIs

#### POST /api/v1/reward
Create a new stock reward for a user.

**Request:**
\`\`\`json
{
  "user_id": "user123",
  "stock_symbol": "RELIANCE",
  "shares": 2.5,
  "reward_type": "onboarding",
  "timestamp": "2024-01-15T10:30:00Z"
}
\`\`\`

**Response:**
\`\`\`json
{
  "success": true,
  "data": {
    "id": 1,
    "event_id": "uuid-here",
    "user_id": "user123",
    "stock_symbol": "RELIANCE",
    "shares": 2.5,
    "reward_type": "onboarding",
    "timestamp": "2024-01-15T10:30:00Z",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
\`\`\`

#### GET /api/v1/today-stocks/{userId}
Get all stock rewards for a user for today.

**Response:**
\`\`\`json
{
  "success": true,
  "data": {
    "user_id": "user123",
    "date": "2024-01-15",
    "stocks": [
      {
        "id": 1,
        "event_id": "uuid-here",
        "stock_symbol": "RELIANCE",
        "shares": 2.5,
        "reward_type": "onboarding",
        "timestamp": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
\`\`\`

#### GET /api/v1/historical-inr/{userId}
Get historical INR values of user's stock rewards.

**Response:**
\`\`\`json
{
  "success": true,
  "data": {
    "user_id": "user123",
    "days": [
      {
        "date": "2024-01-14",
        "inr_value": 6250.75
      },
      {
        "date": "2024-01-13",
        "inr_value": 6180.50
      }
    ]
  }
}
\`\`\`

#### GET /api/v1/stats/{userId}
Get user statistics including today's shares and current portfolio value.

**Response:**
\`\`\`json
{
  "success": true,
  "data": {
    "user_id": "user123",
    "total_shares_today": [
      {
        "stock_symbol": "RELIANCE",
        "total_shares": 2.5
      }
    ],
    "current_inr_value": 6250.75
  }
}
\`\`\`

#### GET /api/v1/portfolio/{userId}
Get complete portfolio with current holdings and valuations.

**Response:**
\`\`\`json
{
  "success": true,
  "data": {
    "user_id": "user123",
    "holdings": [
      {
        "stock_symbol": "RELIANCE",
        "total_shares": 5.0,
        "current_price": 2500.30,
        "inr_value": 12501.50
      }
    ],
    "total_inr_value": 12501.50
  }
}
\`\`\`

### Ledger APIs

#### GET /api/v1/ledger/entries
Get ledger entries with filtering options.

**Query Parameters:**
- `user_id`: Filter by user ID
- `event_id`: Filter by event ID
- `entry_type`: Filter by entry type (STOCK_CREDIT, CASH_DEBIT, etc.)
- `account_type`: Filter by account type (USER_STOCK, COMPANY_CASH, etc.)
- `start_date`: Start date (YYYY-MM-DD)
- `end_date`: End date (YYYY-MM-DD)
- `limit`: Maximum number of entries

#### POST /api/v1/ledger/adjustment
Create an adjustment for a previous reward.

**Request:**
\`\`\`json
{
  "original_event_id": "uuid-here",
  "shares_adjustment": 0.5,
  "reason": "Correction for calculation error"
}
\`\`\`

#### POST /api/v1/ledger/refund
Create a refund for a previous reward.

**Request:**
\`\`\`json
{
  "original_event_id": "uuid-here",
  "reason": "User eligibility revoked"
}
\`\`\`

#### GET /api/v1/ledger/balance/{accountType}
Get account balance for a specific account type.

**Query Parameters:**
- `user_id`: Filter by user ID (optional)
- `stock_symbol`: Filter by stock symbol (optional)

#### GET /api/v1/ledger/integrity
Validate ledger integrity and get a report.

### Admin APIs

#### GET /admin/price-status
Get price updater service status.

#### POST /admin/force-price-update
Force an immediate price update.

#### GET /admin/stale-data
Get information about stale price data.

## Setup Instructions

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher
- Git

### Installation

1. **Clone the repository:**
\`\`\`bash
git clone <repository-url>
cd stocky-backend
\`\`\`

2. **Install dependencies:**
\`\`\`bash
go mod download
\`\`\`

3. **Setup PostgreSQL database:**
\`\`\`bash
createdb assignment
\`\`\`

4. **Configure environment variables:**
Create a `.env` file in the root directory:
\`\`\`env
DATABASE_URL=postgres://postgres:password@localhost/assignment?sslmode=disable
PORT=8080
\`\`\`

5. **Run the application:**
\`\`\`bash
go run main.go
\`\`\`

The server will start on port 8080 and automatically run database migrations.

### Database Setup

The application automatically creates all necessary tables and indexes on startup. To seed initial stock data:

\`\`\`bash
# The application will create the stocks table and you can run the seed script
psql -d assignment -f scripts/seed_stocks.sql
\`\`\`

## Edge Cases Handled

### 1. Duplicate Reward Events
- Each reward event has a unique `event_id` (UUID)
- Database constraints prevent duplicate entries
- Idempotent API design

### 2. Stock Splits, Mergers, Delisting
- Flexible schema supports stock symbol changes
- Adjustment mechanism for handling corporate actions
- Historical data preservation

### 3. Rounding Errors
- Uses `NUMERIC(18,6)` for share quantities
- Uses `NUMERIC(18,4)` for INR amounts
- Proper decimal handling in calculations

### 4. Price API Downtime
- Retry logic with exponential backoff
- Fallback to cached prices
- Stale data detection and alerts
- Manual price update capability

### 5. Adjustments and Refunds
- Complete audit trail for all changes
- Double-entry accounting for adjustments
- Prevents duplicate refunds
- Maintains data integrity

## Scaling Considerations

### Database Optimization
- Proper indexing on frequently queried columns
- Partitioning strategy for large tables (by date)
- Connection pooling for high concurrency

### Caching Strategy
- Redis for frequently accessed stock prices
- Application-level caching for user portfolios
- Cache invalidation on price updates

### Microservices Architecture
- Separate services for rewards, pricing, and ledger
- Event-driven architecture with message queues
- Independent scaling of components

### Monitoring and Observability
- Structured logging with Logrus
- Metrics collection for API performance
- Health checks for all services
- Alerting for system anomalies

## Testing

### Unit Tests
\`\`\`bash
go test ./...
\`\`\`

### Integration Tests
\`\`\`bash
go test -tags=integration ./...
\`\`\`

### Load Testing
Use the provided Postman collection for load testing scenarios.

## Security Considerations

- Input validation on all API endpoints
- SQL injection prevention with parameterized queries
- Rate limiting for API endpoints
- Authentication and authorization (ready for implementation)
- Audit logging for all financial operations

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## License

This project is licensed under the MIT License.
#   s t o c k y - b a c k e n d  
 