package db

import (
	"collaborative-markdown-editor/internal/config"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var AppDb *gorm.DB

func ConnectDb() error {
	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable", 
			config.AppConfig.DBHost,
			config.AppConfig.DBUser,
			config.AppConfig.DBPassword,
			config.AppConfig.DBName,
			config.AppConfig.DBPort,
		)
		
	level := logger.Info
	if config.AppConfig.Environment == "production" {
		level = logger.Error
	}
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:	time.Second,  	// Slow SQL threshold
			LogLevel:       level,   		// Log level
			Colorful:       true,           // Enable color
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: newLogger})

	if err != nil {
		log.Fatalf("error connecting to db %v", err)
		return err
	}
	AppDb = db
	log.Println("Success connecting to db")
	
	return nil
}

func CloseDb() {
	sqlDB, _ := AppDb.DB()
	err := sqlDB.Close()

	if err != nil {
		log.Fatalf("failed to close db %v", err)
	}
	fmt.Println("Closing DB")
}
