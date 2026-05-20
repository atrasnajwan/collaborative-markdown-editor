package notification

import (
	"collaborative-markdown-editor/internal/kafka"
	"collaborative-markdown-editor/internal/sync"
	"collaborative-markdown-editor/internal/worker"
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	kafkaProducer *kafka.KafkaProducer
	workerPool    *worker.WorkerPool
	syncClient    *sync.SyncClient
}

func NewService(kp *kafka.KafkaProducer, wp *worker.WorkerPool, syncClient *sync.SyncClient) *Service {
	return &Service{
		kafkaProducer: kp,
		workerPool:    wp,
		syncClient:    syncClient,
	}
}

type DocUserMessage struct {
	EventID        string `json:"event_id"`
	Type           string `json:"type"`
	DocumentID     uint64 `json:"document_id"`
	AffectedUserID uint64 `json:"affected_user_id"`
	Role           string `json:"role"`
	Timestamp      int64  `json:"timestamp"`
}

type DocMessage struct {
	EventID        string `json:"event_id"`
	Type           string `json:"type"`
	DocumentID     uint64 `json:"document_id"`
	Timestamp      int64  `json:"timestamp"`
}

func (s *Service) NotifyUserRoleChanged(docID, affectedUserID uint64, newRole string) {
	// prioritize kafka
	if s.kafkaProducer != nil {
		message := &DocUserMessage{
			EventID:        uuid.New().String(),
			Type:           "document.role_updated",
			DocumentID:     docID,
			AffectedUserID: affectedUserID,
			Role:           newRole,
			Timestamp:      time.Now().Unix(),
		}
		s.kafkaProducer.SendMessage("notification-events", strconv.FormatUint(docID, 10), message)
		return
	}

	// directly send notification to sync server via worker pool
	s.workerPool.Submit(func(bgCtx context.Context) error {
		// 5s timeout
		timeoutCtx, cancel := context.WithTimeout(bgCtx, 5*time.Second)
		defer cancel()

		// send update to sync-server
		return s.syncClient.UpdateUserPermission(
			timeoutCtx,
			docID,
			affectedUserID,
			newRole,
		)
	})
}

func (s *Service) NotifyDocumentDeleted(docID uint64) {
	// prioritize kafka
	if s.kafkaProducer != nil {
		message := &DocMessage{
			EventID:        uuid.New().String(),
			Type:           "document.deleted",
			DocumentID:     docID,
			Timestamp:      time.Now().Unix(),
		}
		s.kafkaProducer.SendMessage("notification-events", strconv.FormatUint(docID, 10), message)
		return
	}

	// directly send notification to sync server via worker pool
	s.workerPool.Submit(func(bgCtx context.Context) error {
		// 5s timeout
		timeoutCtx, cancel := context.WithTimeout(bgCtx, 5*time.Second)
		defer cancel()

		// send update to sync-server
		return s.syncClient.RemoveDocument(
			timeoutCtx,
			docID,
		)
	})
}
