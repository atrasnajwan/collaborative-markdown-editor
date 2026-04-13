package kafka

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/rs/zerolog/log"

	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/document"
	"collaborative-markdown-editor/internal/worker"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaConsumer struct {
	consumer   *kafka.Consumer
	wp         *worker.WorkerPool
	docService document.Service
	isRunning  bool
}

type KafkaDocMessage struct {
	EventID    string `json:"event_id"`
	Type       string `json:"type"`
	DocumentID uint64 `json:"document_id"`
	UserID     uint64 `json:"user_id"`
	Timestamp  int64  `json:"timestamp"`
	Data       string `json:"data"`
}

func NewKafkaConsumer(workerPool *worker.WorkerPool, docService document.Service) (*KafkaConsumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        config.AppConfig.KafkaBootstrapServers,
		"group.id":                 "document-sync-group",
		"auto.offset.reset":        "earliest",
		"enable.auto.commit":       false,
		"allow.auto.create.topics": true,
	})

	if err != nil {
		return nil, err
	}

	c.SubscribeTopics([]string{"document.sync"}, nil)

	return &KafkaConsumer{consumer: c, wp: workerPool, docService: docService, isRunning: true}, nil
}

func (kc *KafkaConsumer) Start() error {
	for kc.isRunning {
		ev := kc.consumer.Poll(100) // 100 ms
		if ev == nil {
			continue
		}
		switch e := ev.(type) {
		case *kafka.Message:
			kc.wp.Submit(func(bgCtx context.Context) error {
				return kc.processMessage(bgCtx, e)
			})
		case kafka.Error:
			// ERRORS: Handle connection issues
			log.Error().Err(e).Msg("Consumer error")
			if e.IsFatal() {
				kc.isRunning = false
			}
		default:
			log.Debug().Msgf("Ignored %v\n", e)
		}
	}
	return nil
}

func (kc *KafkaConsumer) Close() error {
	kc.isRunning = false
	return kc.consumer.Close()
}

func (kc *KafkaConsumer) processMessage(ctx context.Context, message *kafka.Message) error {
	log.Debug().Msgf("Processing message: Topic=%s, Partition=%d, Key=%s, Value=%s\n",
		*message.TopicPartition.Topic, message.TopicPartition.Partition, string(message.Key), string(message.Value))

	switch *message.TopicPartition.Topic {
	case "document.sync":
		var kafkaDocMsg KafkaDocMessage
		if err := json.Unmarshal(message.Value, &kafkaDocMsg); err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal Kafka Doc message")
			return err
		}
		switch kafkaDocMsg.Type {
		case "document.updated":
			docUpdate, err := base64.StdEncoding.DecodeString(kafkaDocMsg.Data)
			if err != nil {
				log.Error().Err(err).Msg("Failed to decode document update")
				return err
			}
			err = kc.docService.CreateDocumentUpdate(ctx, kafkaDocMsg.DocumentID, kafkaDocMsg.UserID, docUpdate)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create document update")
				return err
			}
			// commit message to kafka
			kc.consumer.CommitMessage(message)
			return nil
		case "document.snapshot":
			docSnapshot, err := base64.StdEncoding.DecodeString(kafkaDocMsg.Data)
			if err != nil {
				log.Error().Err(err).Msg("Failed to decode document snapshot")
				return err
			}
			err = kc.docService.CreateDocumentSnapshot(ctx, kafkaDocMsg.DocumentID, docSnapshot)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create document snapshot")
				return err
			}
			// commit message to kafka
			kc.consumer.CommitMessage(message)
			return nil
		default:
			log.Debug().Msgf("Unknown message type %s in topic %s", kafkaDocMsg.Type, *message.TopicPartition.Topic)
		}
	}
	return nil
}
