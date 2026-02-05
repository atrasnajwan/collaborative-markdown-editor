package main

import (
	"collaborative-markdown-editor/auth"
	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/db"
	"collaborative-markdown-editor/internal/document"
	"collaborative-markdown-editor/internal/sync"
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

	"github.com/gin-contrib/cors"
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
	syncClient := sync.NewSyncClient()
	docService := document.NewService(docRepo, userService, syncClient)
	// Initialize handler
	docHandler := document.NewHandler(docService)
	userHandler := user.NewHandler(userService)

	// Initialize Gin router
	router := gin.Default()

	// cors setting
	corsConfig := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}

	if config.AppConfig.Environment == "development" {
		// Allow all origins in development
		corsConfig.AllowAllOrigins = true
	} else {
		// Restrict origins in production
		corsConfig.AllowOrigins = []string{"https://production-frontend.com"}
	}
	router.Use(cors.New(corsConfig))

	// User routes
	router.POST("/register", userHandler.Register)
	router.POST("/login", userHandler.Login)
	router.DELETE("/logout", auth.AuthMiddleWare(), userHandler.Logout)
	router.GET("/profile", auth.AuthMiddleWare(), userHandler.GetProfile)
	router.GET("/users", auth.AuthMiddleWare(), userHandler.SearchUsers)
	router.POST("/documents", auth.AuthMiddleWare(), docHandler.Create)
	router.GET("/documents", auth.AuthMiddleWare(), docHandler.ShowUserDocuments)
	router.GET("/documents/:id", auth.AuthMiddleWare(), docHandler.ShowDocument)
	router.DELETE("/documents/:id", auth.AuthMiddleWare(), docHandler.DeleteDocument)
	router.GET("/documents/:id/collaborators", auth.AuthMiddleWare(), docHandler.ListCollaborators)
	router.POST("/documents/:id/collaborators", auth.AuthMiddleWare(), docHandler.AddCollaborator)
	router.PUT("/documents/:id/collaborators", auth.AuthMiddleWare(), docHandler.ChangeCollaboratorRole)
	router.DELETE("/documents/:id/collaborators/:userId", auth.AuthMiddleWare(), docHandler.RemoveCollaborator)

	// internal use routes
	router.GET("/internal/documents/:id/permission", auth.InternalAuthMiddleware(config.AppConfig.InternalSecret), docHandler.ShowUserRole)
	router.GET("/internal/documents/:id/last-state", auth.InternalAuthMiddleware(config.AppConfig.InternalSecret), docHandler.ShowDocumentState)
	router.POST("/internal/documents/:id/update", auth.InternalAuthMiddleware(config.AppConfig.InternalSecret), docHandler.CreateUpdate)
	router.POST("/internal/documents/:id/snapshot", auth.InternalAuthMiddleware(config.AppConfig.InternalSecret), docHandler.CreateSnapshot)

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
