package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nexus-rmm/backend/internal/api"
	"github.com/nexus-rmm/backend/internal/auth"
	"github.com/nexus-rmm/backend/internal/config"
	"github.com/nexus-rmm/backend/internal/storage"
)

func main() {
	// Parse command line flags
	mode := flag.String("mode", "api", "Run mode: api, gateway, migrate")
	migrateCmd := flag.String("migrate", "", "Migration command: up, down, create")
	migrateName := flag.String("name", "", "Migration name (for create command)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := storage.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Handle migration commands
	if *mode == "migrate" {
		ctx := context.Background()
		switch *migrateCmd {
		case "up":
			if err := storage.MigrateUp(ctx, db.DB); err != nil {
				log.Fatalf("Migration up failed: %v", err)
			}
			log.Println("Migrations applied successfully")
		case "down":
			if err := storage.MigrateDown(ctx, db.DB); err != nil {
				log.Fatalf("Migration down failed: %v", err)
			}
			log.Println("Migration rolled back successfully")
		case "create":
			if *migrateName == "" {
				log.Fatal("Migration name required for create command")
			}
			if err := storage.CreateMigration(ctx, db.DB, *migrateName); err != nil {
				log.Fatalf("Failed to create migration: %v", err)
			}
			log.Printf("Migration '%s' created successfully", *migrateName)
		default:
			log.Fatalf("Unknown migration command: %s", *migrateCmd)
		}
		return
	}

	// Initialize database schema
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	schemaErr := storage.InitializeSchema(ctx, db.DB)
	cancel()
	if schemaErr != nil {
		log.Fatalf("Failed to initialize schema: %v", schemaErr)
	}

	authService := auth.NewAuthService(auth.Config{
		JWTSecret:          cfg.Auth.JWTSecret,
		JWTIssuer:          cfg.Auth.JWTIssuer,
		AccessTokenTTL:     cfg.Auth.AccessTokenTTL,
		RefreshTokenTTL:    cfg.Auth.RefreshTokenTTL,
		MFAIssuer:          cfg.Auth.MFAIssuer,
		BcryptCost:         cfg.Auth.BcryptCost,
		MaxLoginAttempts:   cfg.Auth.MaxLoginAttempts,
		LockoutDuration:    cfg.Auth.LockoutDuration,
		PasswordMinLength:  cfg.Auth.PasswordMinLength,
		RequireMFA:         cfg.Auth.RequireMFA,
	})

	// Handle gateway mode
	if *mode == "gateway" {
		runGateway(cfg, db, authService)
		return
	}

	// Default: run API server
	runAPIServer(cfg, db, authService)
}

func runAPIServer(cfg *config.Config, db *storage.Database, authService *auth.AuthService) {
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now()})
	})

	router.GET("/api/v1/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"version": "1.0.0",
			"name":    "Nexus RMM API",
		})
	})

	v1 := router.Group("/api/v1")
	{
		authHandler := api.NewAuthHandler(db, authService)
		authHandler.RegisterRoutes(v1)

		protected := v1.Group("/")
		protected.Use(api.AuthMiddleware(authService))
		protected.Use(api.TenantIsolationMiddleware())
		{
			deviceHandler := api.NewDeviceHandler(db)
			deviceHandler.RegisterRoutes(protected)

			orgHandler := api.NewOrganizationHandler(db)
			orgHandler.RegisterRoutes(protected)

			remoteHandler := api.NewRemoteHandler(db)
			remoteHandler.RegisterRoutes(protected)
		}
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Starting API server on port %d", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func runGateway(cfg *config.Config, db *storage.Database, authService *auth.AuthService) {
	router := gin.Default()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now(), "mode": "gateway"})
	})

	// Gateway routes would go here
	v1 := router.Group("/api/v1")
	{
		remoteHandler := api.NewRemoteHandler(db)
		remoteHandler.RegisterRoutes(v1)
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Remote.GatewayPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Starting Gateway server on port %d", cfg.Remote.GatewayPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Gateway server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gateway...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Gateway forced to shutdown: %v", err)
	}

	log.Println("Gateway exited")
}