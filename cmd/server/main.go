package main

import (
	"collaborative-markdown-editor/auth"
	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/db"
	"collaborative-markdown-editor/internal/document"
	"collaborative-markdown-editor/internal/user"
	"collaborative-markdown-editor/redis"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Connect to database
	db.ConnectDb()
	defer db.CloseDb()

	// Migrate database schema
	db.Migrate()

	// Seed database with initial data (for development)
	db.SeedData()

	// Initialize Redis
	redis.InitRedis()

	// Initialize repository
	userRepo := user.NewRepository(db.AppDb)
	docRepo := document.NewRepository(db.AppDb)
	// Initialize service
	userService := user.NewService(userRepo)
	docService := document.NewService(docRepo)
	// Initialize handler
	docHandler := document.NewHandler(docService)
	userHandler := user.NewHandler(userService)

	// Initialize Gin router
	router := gin.Default()

	// User routes
	router.POST("/register", userHandler.Register)
	router.POST("/login", userHandler.Login)
	router.DELETE("/logout", auth.AuthMiddleWare(), userHandler.Logout)
	router.GET("/profile", auth.AuthMiddleWare(), userHandler.GetProfile)
	router.POST("/documents", auth.AuthMiddleWare(), docHandler.Create)

	// Server configuration
	serverPort := config.AppConfig.ServerPort
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", serverPort),
		Handler: router.Handler(),
	}

	// Start server
	go func() {
		log.Printf("Server listening on port %s", serverPort)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Println("Server shutdown error:", err)
	}

	<-ctx.Done()
	log.Println("Server shutdown complete")
}
