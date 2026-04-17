package event

import (
	"collaborative-markdown-editor/internal/document"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/rs/zerolog/log"
)

type Message struct {
	EventID string `json:"event_id"`
}

type Service struct {
	repository EventRepository
	docService document.Service
}

func NewService(repo EventRepository, docService document.Service) Service {
	return Service{repository: repo}
}

func (s Service) CanProcess(ctx context.Context, msg []byte) bool {
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

type DocumentMessage struct {
	EventID    string `json:"event_id"`
	Type       string `json:"type"`
	DocumentID uint64 `json:"document_id"`
	UserID     uint64 `json:"user_id"`
	Timestamp  int64  `json:"timestamp"`
	Data       string `json:"data"`
}

func (s Service) ProcessDocumentEvent(ctx context.Context, message *kafka.Message) error {
	var docMessage DocumentMessage
	if err := json.Unmarshal(message.Value, &docMessage); err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal Kafka Doc message")
		return err
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
