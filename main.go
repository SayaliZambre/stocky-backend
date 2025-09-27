package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"stocky/internal/api"
	"stocky/internal/config"
	"stocky/internal/database"
	"stocky/internal/services"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logrus.Warn("No .env file found")
	}

	// Initialize configuration
	cfg := config.Load()

	// Setup logging
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// Initialize database
	db, err := database.Initialize(cfg.DatabaseURL)
	if err != nil {
		logrus.Fatal("Failed to initialize database: ", err)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		logrus.Fatal("Failed to run migrations: ", err)
	}

	// Initialize services
	stockService := services.NewStockService()
	rewardService := services.NewRewardService(db)
	portfolioService := services.NewPortfolioService(db, stockService)
	ledgerService := services.NewLedgerService(db)
	
	priceUpdater := services.NewPriceUpdaterService(db, stockService)
	priceUpdater.StartPriceUpdater()

	// Initialize router
	router := gin.Default()
	
	// Setup API routes
	api.SetupRoutes(router, rewardService, portfolioService, ledgerService)
	
	api.SetupAdminRoutes(router, priceUpdater)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logrus.Info("Shutting down gracefully...")
		priceUpdater.StopPriceUpdater()
		os.Exit(0)
	}()

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logrus.Infof("Starting server on port %s", port)
	log.Fatal(router.Run(":" + port))
}
