package db

import (
	"collaborative-markdown-editor/internal/config"
	"fmt"
	stdlog "log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	log "github.com/rs/zerolog/log"
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
		stdlog.New(os.Stdout, "\r\n", stdlog.LstdFlags), // io writer for gorm
		logger.Config{
			SlowThreshold: time.Second, // Slow SQL threshold
			LogLevel:      level,       // Log level
			Colorful:      true,        // Enable color
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: newLogger})

	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to db")
		return err
	}
	AppDb = db
	log.Info().Msg("Success connecting to db")

	return nil
}

func CloseDb() {
	sqlDB, _ := AppDb.DB()
	err := sqlDB.Close()

	if err != nil {
		log.Fatal().Err(err).Msg("failed to close db")
	}
	log.Info().Msg("Closing DB")
}
