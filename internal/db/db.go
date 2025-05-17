package db

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectDb() (*gorm.DB, error) {
	fmt.Println(os.Getenv("PORT"))
	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v sslmode=disable", os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_PORT"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("error connecting to db %v", err)
		return nil, err
	}

	fmt.Printf("Success connecting to db %v\n", os.Getenv("DB_NAME"))
	return db, nil
}

func CloseDb(appDb *gorm.DB) {
	sqlDB, _ := appDb.DB()
	err := sqlDB.Close()

	if err != nil {
		log.Fatalf("failed to close db %v", err)
	}
	fmt.Println("Closing DB")
}
