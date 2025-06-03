package db

import (
	"collaborative-markdown-editor/internal/config"
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

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
