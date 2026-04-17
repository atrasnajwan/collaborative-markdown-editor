package event

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/rs/zerolog/log"
)

type Message struct {
	EventID    string `json:"event_id"`
}

type EventService struct {
	repository EventRepository
}

func NewService(repo EventRepository) EventService {
	return EventService{repository: repo}
}

func (s EventService) CanProcess(ctx context.Context, msg []byte) bool {
	var message Message
	if err := json.Unmarshal(msg, &message); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal message")
		return false
	}

	err := s.repository.Create(ctx, message)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
				log.Debug().Msgf("Event already processed %s", message.EventID)
                return false
        }
		log.Error().Err(err).Msg("Failed to insert message to DB")
		return false
	}
	return true
}