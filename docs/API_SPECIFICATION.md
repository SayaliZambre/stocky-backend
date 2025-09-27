# Stocky API Specification

## Overview

The Stocky API provides comprehensive endpoints for managing stock rewards, portfolio tracking, and financial ledger operations. All endpoints return JSON responses with consistent error handling.

## Base URL

\`\`\`
http://localhost:8080
\`\`\`

## Response Format

All API responses follow this structure:

\`\`\`json
{
  "success": boolean,
  "data": object | array,
  "error": string (only on failure),
  "details": string (only on failure)
}
\`\`\`

## Error Codes

- `400 Bad Request`: Invalid request payload or parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server-side error

## Authentication

Currently, the API does not require authentication. In production, implement JWT or API key authentication.

## Rate Limiting

No rate limiting is currently implemented. Consider adding rate limiting for production use.

## Endpoints

### Health Check

#### GET /health

Check if the service is running.

**Response:**
\`\`\`json
{
  "status": "healthy"
}
\`\`\`

---

## Core Reward APIs

### Create Stock Reward

#### POST /api/v1/reward

Award stock shares to a user.

**Request Body:**
\`\`\`json
{
  "user_id": "string (required)",
  "stock_symbol": "string (required)",
  "shares": "number (required, > 0)",
  "reward_type": "string (required)",
  "timestamp": "string (optional, ISO 8601)"
}
\`\`\`

**Example:**
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
    "event_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "user123",
    "stock_symbol": "RELIANCE",
    "shares": 2.5,
    "reward_type": "onboarding",
    "timestamp": "2024-01-15T10:30:00Z",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
\`\`\`

### Get Today's Stock Rewards

#### GET /api/v1/today-stocks/{userId}

Retrieve all stock rewards for a user for the current date.

**Path Parameters:**
- `userId` (string, required): User identifier

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
        "event_id": "550e8400-e29b-41d4-a716-446655440000",
        "user_id": "user123",
        "stock_symbol": "RELIANCE",
        "shares": 2.5,
        "reward_type": "onboarding",
        "timestamp": "2024-01-15T10:30:00Z",
        "created_at": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
\`\`\`

### Get Historical INR Values

#### GET /api/v1/historical-inr/{userId}

Get the INR value of user's stock rewards for all past days (excluding today).

**Path Parameters:**
- `userId` (string, required): User identifier

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

### Get User Statistics

#### GET /api/v1/stats/{userId}

Get user statistics including today's shares and current portfolio value.

**Path Parameters:**
- `userId` (string, required): User identifier

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
      },
      {
        "stock_symbol": "TCS",
        "total_shares": 1.0
      }
    ],
    "current_inr_value": 12501.50
  }
}
\`\`\`

### Get Portfolio

#### GET /api/v1/portfolio/{userId}

Get complete user portfolio with current holdings and valuations.

**Path Parameters:**
- `userId` (string, required): User identifier

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
      },
      {
        "stock_symbol": "TCS",
        "total_shares": 2.0,
        "current_price": 3800.25,
        "inr_value": 7600.50
      }
    ],
    "total_inr_value": 20102.00
  }
}
\`\`\`

---

## Ledger APIs

### Get Ledger Entries

#### GET /api/v1/ledger/entries

Retrieve ledger entries with optional filtering.

**Query Parameters:**
- `user_id` (string, optional): Filter by user ID
- `event_id` (string, optional): Filter by event ID
- `entry_type` (string, optional): Filter by entry type
- `account_type` (string, optional): Filter by account type
- `start_date` (string, optional): Start date (YYYY-MM-DD)
- `end_date` (string, optional): End date (YYYY-MM-DD)
- `limit` (integer, optional): Maximum number of entries

**Entry Types:**
- `STOCK_CREDIT`: Stock credited to user
- `STOCK_DEBIT`: Stock debited from user
- `CASH_DEBIT`: Cash debited from company
- `CASH_CREDIT`: Cash credited to company
- `FEE_DEBIT`: Fees debited from company

**Account Types:**
- `USER_STOCK`: User stock holdings
- `COMPANY_CASH`: Company cash account
- `COMPANY_FEES`: Company fees account

**Response:**
\`\`\`json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "event_id": "550e8400-e29b-41d4-a716-446655440000",
      "entry_type": "STOCK_CREDIT",
      "account_type": "USER_STOCK",
      "user_id": "user123",
      "stock_symbol": "RELIANCE",
      "amount": null,
      "shares": 2.5,
      "description": "Stock reward: 2.500000 shares of RELIANCE",
      "timestamp": "2024-01-15T10:30:00Z",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
\`\`\`

### Create Adjustment

#### POST /api/v1/ledger/adjustment

Create an adjustment for a previous reward event.

**Request Body:**
\`\`\`json
{
  "original_event_id": "string (required)",
  "shares_adjustment": "number (required)",
  "reason": "string (required)"
}
\`\`\`

**Example:**
\`\`\`json
{
  "original_event_id": "550e8400-e29b-41d4-a716-446655440000",
  "shares_adjustment": 0.5,
  "reason": "Correction for calculation error"
}
\`\`\`

**Response:**
\`\`\`json
{
  "success": true,
  "message": "Adjustment created successfully"
}
\`\`\`

### Create Refund

#### POST /api/v1/ledger/refund

Create a refund for a previous reward event.

**Request Body:**
\`\`\`json
{
  "original_event_id": "string (required)",
  "reason": "string (required)"
}
\`\`\`

**Example:**
\`\`\`json
{
  "original_event_id": "550e8400-e29b-41d4-a716-446655440000",
  "reason": "User eligibility revoked"
}
\`\`\`

**Response:**
\`\`\`json
{
  "success": true,
  "message": "Refund created successfully"
}
\`\`\`

### Get Account Balance

#### GET /api/v1/ledger/balance/{accountType}

Get balance for a specific account type.

**Path Parameters:**
- `accountType` (string, required): Account type to query

**Query Parameters:**
- `user_id` (string, optional): Filter by user ID
- `stock_symbol` (string, optional): Filter by stock symbol

**Response:**
\`\`\`json
{
  "success": true,
  "data": {
    "account_type": "USER_STOCK",
    "user_id": "user123",
    "stock_symbol": "RELIANCE",
    "cash_balance": 0.0,
    "stock_balance": 5.0,
    "entry_count": 3,
    "last_updated": "2024-01-15T10:30:00Z"
  }
}
\`\`\`

### Validate Ledger Integrity

#### GET /api/v1/ledger/integrity

Validate the integrity of the double-entry ledger system.

**Response:**
\`\`\`json
{
  "success": true,
  "data": {
    "check_time": "2024-01-15T10:30:00Z",
    "is_valid": true,
    "issues": []
  }
}
\`\`\`

If issues are found:
\`\`\`json
{
  "success": true,
  "data": {
    "check_time": "2024-01-15T10:30:00Z",
    "is_valid": false,
    "issues": [
      "Event 550e8400-e29b-41d4-a716-446655440000 has unbalanced amount: 0.0100",
      "User user123 stock RELIANCE: reward_shares=5.000000, ledger_shares=4.999999"
    ]
  }
}
\`\`\`

---

## Admin APIs

### Get Price Status

#### GET /admin/price-status

Get the status of the price updater service.

**Response:**
\`\`\`json
{
  "success": true,
  "data": {
    "is_running": true,
    "last_update": "2024-01-15T10:00:00Z"
  }
}
\`\`\`

### Force Price Update

#### POST /admin/force-price-update

Trigger an immediate price update for all stocks.

**Response:**
\`\`\`json
{
  "success": true,
  "message": "Price update completed successfully"
}
\`\`\`

### Get Stale Data Status

#### GET /admin/stale-data

Get information about stale price data.

**Response:**
\`\`\`json
{
  "success": true,
  "data": {
    "total_symbols": 10,
    "stale_symbols": ["RELIANCE", "TCS"],
    "stale_count": 2,
    "last_check": "2024-01-15T10:30:00Z",
    "stale_threshold": "2024-01-15T08:30:00Z"
  }
}
\`\`\`

## Stock Symbols

The system supports the following Indian stock symbols:

- `RELIANCE` - Reliance Industries Limited
- `TCS` - Tata Consultancy Services Limited
- `INFY` - Infosys Limited
- `HDFCBANK` - HDFC Bank Limited
- `ICICIBANK` - ICICI Bank Limited
- `HINDUNILVR` - Hindustan Unilever Limited
- `ITC` - ITC Limited
- `SBIN` - State Bank of India
- `BHARTIARTL` - Bharti Airtel Limited
- `KOTAKBANK` - Kotak Mahindra Bank Limited

Additional stocks can be added to the `stocks` table as needed.
