package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"stocky/internal/services"
)

func SetupLedgerRoutes(router *gin.Engine, ledgerService *services.LedgerService) {
	ledger := router.Group("/api/v1/ledger")
	{
		ledger.GET("/entries", GetLedgerEntries(ledgerService))
		ledger.POST("/adjustment", CreateAdjustment(ledgerService))
		ledger.POST("/refund", CreateRefund(ledgerService))
		ledger.GET("/balance/:accountType", GetAccountBalance(ledgerService))
		ledger.GET("/integrity", ValidateLedgerIntegrity(ledgerService))
	}
}

func GetLedgerEntries(ledgerService *services.LedgerService) gin.HandlerFunc {
	return func(c *gin.Context) {
		filter := &services.LedgerFilter{
			UserID:      c.Query("user_id"),
			EventID:     c.Query("event_id"),
			EntryType:   c.Query("entry_type"),
			AccountType: c.Query("account_type"),
		}

		if startDate := c.Query("start_date"); startDate != "" {
			if parsed, err := time.Parse("2006-01-02", startDate); err == nil {
				filter.StartDate = parsed
			}
		}

		if endDate := c.Query("end_date"); endDate != "" {
			if parsed, err := time.Parse("2006-01-02", endDate); err == nil {
				filter.EndDate = parsed
			}
		}

		if limit := c.Query("limit"); limit != "" {
			if parsed, err := strconv.Atoi(limit); err == nil {
				filter.Limit = parsed
			}
		}

		entries, err := ledgerService.GetLedgerEntries(filter)
		if err != nil {
			logrus.WithError(err).Error("Failed to get ledger entries")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get ledger entries",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": entries,
		})
	}
}

func CreateAdjustment(ledgerService *services.LedgerService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.AdjustmentRequest
		
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request payload",
				"details": err.Error(),
			})
			return
		}

		if err := ledgerService.CreateAdjustment(&req); err != nil {
			logrus.WithError(err).Error("Failed to create adjustment")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create adjustment",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "Adjustment created successfully",
		})
	}
}

func CreateRefund(ledgerService *services.LedgerService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req services.RefundRequest
		
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request payload",
				"details": err.Error(),
			})
			return
		}

		if err := ledgerService.CreateRefund(&req); err != nil {
			logrus.WithError(err).Error("Failed to create refund")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create refund",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"message": "Refund created successfully",
		})
	}
}

func GetAccountBalance(ledgerService *services.LedgerService) gin.HandlerFunc {
	return func(c *gin.Context) {
		accountType := c.Param("accountType")
		userID := c.Query("user_id")
		stockSymbol := c.Query("stock_symbol")

		var userIDPtr, stockSymbolPtr *string
		if userID != "" {
			userIDPtr = &userID
		}
		if stockSymbol != "" {
			stockSymbolPtr = &stockSymbol
		}

		balance, err := ledgerService.GetAccountBalance(accountType, userIDPtr, stockSymbolPtr)
		if err != nil {
			logrus.WithError(err).Error("Failed to get account balance")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get account balance",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": balance,
		})
	}
}

func ValidateLedgerIntegrity(ledgerService *services.LedgerService) gin.HandlerFunc {
	return func(c *gin.Context) {
		report, err := ledgerService.ValidateLedgerIntegrity()
		if err != nil {
			logrus.WithError(err).Error("Failed to validate ledger integrity")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to validate ledger integrity",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": report,
		})
	}
}
