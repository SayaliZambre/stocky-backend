package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"stocky/internal/models"
	"stocky/internal/services"
)

func CreateReward(rewardService *services.RewardService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.RewardRequest
		
		if err := c.ShouldBindJSON(&req); err != nil {
			logrus.WithError(err).Error("Invalid request payload")
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request payload",
				"details": err.Error(),
			})
			return
		}

		// Set timestamp if not provided
		if req.Timestamp.IsZero() {
			req.Timestamp = time.Now()
		}

		// Validate stock symbol format
		if len(req.StockSymbol) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Stock symbol is required",
			})
			return
		}

		// Validate shares amount
		if req.Shares <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Shares must be greater than 0",
			})
			return
		}

		reward, err := rewardService.CreateReward(&req)
		if err != nil {
			logrus.WithError(err).Error("Failed to create reward")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to create reward",
				"details": err.Error(),
			})
			return
		}

		logrus.WithFields(logrus.Fields{
			"user_id": req.UserID,
			"stock_symbol": req.StockSymbol,
			"shares": req.Shares,
			"event_id": reward.EventID,
		}).Info("Reward created successfully")

		c.JSON(http.StatusCreated, gin.H{
			"success": true,
			"data": reward,
		})
	}
}

func GetTodayStocks(rewardService *services.RewardService) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("userId")
		
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "User ID is required",
			})
			return
		}

		response, err := rewardService.GetTodayStocks(userID)
		if err != nil {
			logrus.WithError(err).Error("Failed to get today's stocks")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get today's stocks",
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
