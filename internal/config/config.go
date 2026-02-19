package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server configuration
	ServerPort  string
	Environment string

	// Database configuration
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis configuration
	RedisAddress string
	RedisPollSize int

	// JWT configuration
	JWTSecret string

	// Sync server config
	SyncServerAddress string
	SyncServerSecret string

	// internal secret used for communication between server
	InternalSecret string

	FrontendAddress string
}

// Global application configuration
var AppConfig Config

// LoadConfig loads configuration from environment variables
func LoadConfig() {
	// Find .env file
	envPath := ".env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// Try to find .env in parent directories
		envPath = filepath.Join("..", ".env")
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			envPath = filepath.Join("..", "..", ".env")
		}
	}

	// Load .env file if it exists
	if _, err := os.Stat(envPath); err == nil {
		if err := godotenv.Load(envPath); err != nil {
			log.Printf("Warning: Error loading .env file: %v\n", err)
		}
	}

	// Load configuration from environment variables
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = generateRandomSecret(32) // Generate a 32-byte random secret if not declared
		log.Println("Generated random JWT secret")
	}

	AppConfig = Config{
		ServerPort:        	getEnv("PORT", "8080"),
		Environment:       	getEnv("ENV", "development"),
		DBHost:            	getEnv("DB_HOST", "localhost"),
		DBPort:            	getEnv("DB_PORT", "5432"),
		DBUser:            	getEnv("DB_USER", "postgres"),
		DBPassword:        	getEnv("DB_PASSWORD", "postgres"),
		DBName:            	getEnv("DB_NAME", "markdown_editor"),
		RedisAddress:      	getEnv("REDIS_ADDRESS", "localhost:6379"),
		RedisPollSize:      getEnv("REDIS_POOL_SIZE", 10),
		SyncServerAddress: 	getEnv("SYNC_ADDRESS", "http://localhost:8787"),
		SyncServerSecret:  	getEnv("SYNC_SECRET", "collab-sync-secret"),
		JWTSecret:         	jwtSecret,
		InternalSecret:    	getEnv("INTERNAL_SECRET", "collab-internal-secret"),
		FrontendAddress:   	getEnv("FRONTEND_ADDRESS", "https://production-frontend.com"),
	}
}

// gets an environment variable or returns a default value
func getEnv[T any](key string, defaultValue T) T {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	var result any
	switch any(defaultValue).(type) {
	case string:
		result = value
	case int:
		i, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		result = i
	case bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		result = b
	default:
		return defaultValue
	}

	return result.(T)
}

// generateRandomSecret generates a random secret of the specified length
func generateRandomSecret(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	secret := make([]byte, length)
	for i := range secret {
		secret[i] = charset[random(len(charset))]
	}
	return string(secret)
}

// random returns a random integer between 0 and n-1
func random(n int) int {
	return int(time.Now().UnixNano()) % n
}
