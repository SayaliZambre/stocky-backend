package api

import (
	"github.com/gin-gonic/gin"
	"stocky/internal/services"
)

func SetupRoutes(
	router *gin.Engine,
	rewardService *services.RewardService,
	portfolioService *services.PortfolioService,
	ledgerService *services.LedgerService,
) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Reward endpoints
		v1.POST("/reward", CreateReward(rewardService))
		v1.GET("/today-stocks/:userId", GetTodayStocks(rewardService))
		v1.GET("/historical-inr/:userId", GetHistoricalINR(portfolioService))
		v1.GET("/stats/:userId", GetStats(portfolioService))
		v1.GET("/portfolio/:userId", GetPortfolio(portfolioService))
	}

	// Setup ledger routes
	SetupLedgerRoutes(router, ledgerService)
}
