package main

import (
	"collaborative-markdown-editor/internal/db"
	"log"

	"github.com/joho/godotenv"
)

func init() {
    if err := godotenv.Load("../../.env"); err != nil {
        log.Fatal("Error loading .env file\n", err)
    }
}

func main() {
	database, _ := db.ConnectDb()
	defer db.CloseDb(database)

	db.Migrate(database)
}
