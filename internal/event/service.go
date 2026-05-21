package event

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/rs/zerolog/log"
)

type DocProvider interface {
	CreateDocumentUpdate(ctx context.Context, id uint64, userID uint64, content []byte) error
	CreateDocumentSnapshot(ctx context.Context, docID uint64, state []byte) error
}
type Service struct {
	repository EventRepository
	docService DocProvider
}

func NewService(repo EventRepository, docService DocProvider) *Service {
	return &Service{repository: repo, docService: docService}
}

func (s *Service) canProcess(ctx context.Context, eventID string) bool {
	err := s.repository.Create(ctx, eventID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			log.Debug().Msgf("Event already processed %s", eventID)
			return false
		}
		log.Error().Err(err).Msg("Failed to insert message to DB")
		return false
	}
	return true
}

type DocumentMessage struct {
	EventID    string `json:"event_id"`
	Type       string `json:"type"`
	DocumentID uint64 `json:"document_id"`
	UserID     uint64 `json:"user_id"`
	Timestamp  int64  `json:"timestamp"`
	Data       string `json:"data"`
}

func (s *Service) ProcessDocumentEvent(ctx context.Context, message *kafka.Message) error {
	var docMessage DocumentMessage
	if err := json.Unmarshal(message.Value, &docMessage); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal Kafka Doc message")
		return err
	}

	if canProcess := s.canProcess(ctx, docMessage.EventID); !canProcess {
		log.Debug().Msgf("Can't process event %s", docMessage.EventID)
		return errors.New("Can't Process event")
	}

	switch docMessage.Type {
	case "document.updated":
		docUpdate, err := base64.StdEncoding.DecodeString(docMessage.Data)
		if err != nil {
			log.Error().Err(err).Msg("Failed to decode document update")
			return err
		}
		err = s.docService.CreateDocumentUpdate(ctx, docMessage.DocumentID, docMessage.UserID, docUpdate)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create document update")
			return err
		}
	case "document.snapshot":
		docSnapshot, err := base64.StdEncoding.DecodeString(docMessage.Data)
		if err != nil {
			log.Error().Err(err).Msg("Failed to decode document snapshot")
			return err
		}
		err = s.docService.CreateDocumentSnapshot(ctx, docMessage.DocumentID, docSnapshot)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create document snapshot")
			return err
		}
	default:
		log.Debug().Msgf("Unknown message type %s in topic %s", docMessage.Type, *message.TopicPartition.Topic)
	}
	return nil
}

