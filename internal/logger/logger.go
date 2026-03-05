package logger

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init configures the global zerolog logger based on the application environment. 
func Init(env string) {
	// output timestamps in a readable format
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02T15:04:05Z07:00",
	}

	base := zerolog.New(writer).With().Timestamp().Logger()

	switch strings.ToLower(env) {
	case "development", "dev", "":
		base = base.Level(zerolog.DebugLevel)
	default:
		base = base.Level(zerolog.InfoLevel)
	}

	log.Logger = base
}
