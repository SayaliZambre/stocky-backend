package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"stocky/internal/services"
)

// Admin endpoints for price management
func SetupAdminRoutes(router *gin.Engine, priceUpdater *services.PriceUpdaterService) {
	admin := router.Group("/admin")
	{
		admin.GET("/price-status", GetPriceStatus(priceUpdater))
		admin.POST("/force-price-update", ForcePriceUpdate(priceUpdater))
		admin.GET("/stale-data", GetStaleDataStatus(priceUpdater))
	}
}

func GetPriceStatus(priceUpdater *services.PriceUpdaterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := gin.H{
			"is_running":    priceUpdater.IsRunning(),
			"last_update":   priceUpdater.GetLastUpdateTime(),
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    status,
		})
	}
}

func ForcePriceUpdate(priceUpdater *services.PriceUpdaterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := priceUpdater.ForceUpdate(); err != nil {
			logrus.WithError(err).Error("Force price update failed")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update prices",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Price update completed successfully",
		})
	}
}

func GetStaleDataStatus(priceUpdater *services.PriceUpdaterService) gin.HandlerFunc {
	return func(c *gin.Context) {
		status, err := priceUpdater.GetStaleDataStatus()
		if err != nil {
			logrus.WithError(err).Error("Failed to get stale data status")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get stale data status",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    status,
		})
	}
}
