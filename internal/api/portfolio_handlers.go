package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"stocky/internal/services"
)

func GetHistoricalINR(portfolioService *services.PortfolioService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("userId")
		
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "User ID is required",
			})
			return
		}

		response, err := portfolioService.GetHistoricalINR(userID)
		if err != nil {
			logrus.WithError(err).Error("Failed to get historical INR")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get historical INR",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": response,
		})
	}
}

func GetStats(portfolioService *services.PortfolioService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("userId")
		
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "User ID is required",
			})
			return
		}

		response, err := portfolioService.GetStats(userID)
		if err != nil {
			logrus.WithError(err).Error("Failed to get user stats")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get user stats",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": response,
		})
	}
}

func GetPortfolio(portfolioService *services.PortfolioService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("userId")
		
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "User ID is required",
			})
			return
		}

		response, err := portfolioService.GetPortfolio(userID)
		if err != nil {
			logrus.WithError(err).Error("Failed to get portfolio")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get portfolio",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": response,
		})
	}
}
