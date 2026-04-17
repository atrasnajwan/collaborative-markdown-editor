package kafka

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/rs/zerolog/log"

	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/document"
	"collaborative-markdown-editor/internal/event"
	"collaborative-markdown-editor/internal/worker"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaConsumer struct {
	consumer     *kafka.Consumer
	wp           *worker.WorkerPool
	eventService event.EventService
	docService   document.Service
	workers      map[int32]chan *kafka.Message
	isRunning    bool
}

type KafkaDocMessage struct {
	EventID    string `json:"event_id"`
	Type       string `json:"type"`
	DocumentID uint64 `json:"document_id"`
	UserID     uint64 `json:"user_id"`
	Timestamp  int64  `json:"timestamp"`
	Data       string `json:"data"`
}

func NewKafkaConsumer(workerPool *worker.WorkerPool, eventService event.EventService, groupID string, topics []string, docService document.Service) (*KafkaConsumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        config.AppConfig.KafkaBootstrapServers,
		"group.id":                 groupID,
		"auto.offset.reset":        "earliest",
		"enable.auto.commit":       false,
		"allow.auto.create.topics": true,
	})

	if err != nil {
		return nil, err
	}

	c.SubscribeTopics(topics, nil)

	return &KafkaConsumer{
		consumer:     c,
		wp:           workerPool,
		eventService: eventService,
		docService:   docService,
		workers:      map[int32]chan *kafka.Message{},
		isRunning:    true,
	}, nil
}

func (kc *KafkaConsumer) Start() error {
	for kc.isRunning {
		ev := kc.consumer.Poll(100) // 100 ms
		if ev == nil {
			continue
		}
		switch e := ev.(type) {
		case *kafka.Message:
			partition := e.TopicPartition.Partition

			if kc.workers[partition] == nil {
				kc.workers[partition] = make(chan *kafka.Message, 100)

				go kc.startPartitionWorker(partition, kc.workers[partition])
			}

			kc.workers[partition] <- e
		case kafka.Error:
			// ERRORS: Handle connection issues
			log.Error().Err(e).Msg("Consumer error")
			if e.IsFatal() {
				kc.Close()
			}
		default:
			log.Debug().Msgf("Ignored %v\n", e)
		}
	}
	return nil
}

func (kc *KafkaConsumer) startPartitionWorker(
	partition int32,
	ch chan *kafka.Message,
) {
	for msg := range ch {
		bgCtx := context.Background()
		canProcess := kc.eventService.CanProcess(bgCtx, msg.Value)
		if !canProcess {
			log.Debug().Msgf("Can't process message %v", msg.Value)
			continue
		}

		err := kc.processMessage(bgCtx, msg)

		if err != nil {
			log.Error().Err(err).Msgf("Failed on processing partition %d", partition)
			continue
		}

		_, err = kc.consumer.CommitMessage(msg)
		if err != nil {
			log.Error().Err(err).Msg("Commit failed")
		}
	}
}

func (kc *KafkaConsumer) Close() error {
	kc.isRunning = false
	for _, ch := range kc.workers {
		close(ch)
	}
	return kc.consumer.Close()
}

func (kc *KafkaConsumer) processMessage(ctx context.Context, message *kafka.Message) error {
	log.Debug().Msgf("Processing message: Topic=%s, Partition=%d, Key=%s, Value=%s\n",
		*message.TopicPartition.Topic, message.TopicPartition.Partition, string(message.Key), string(message.Value))

	switch *message.TopicPartition.Topic {
	case "document.events":
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
			return nil
		default:
			log.Debug().Msgf("Unknown message type %s in topic %s", kafkaDocMsg.Type, *message.TopicPartition.Topic)
		}
	default:
		log.Debug().Msgf("Unknown topic %s", *message.TopicPartition.Topic)
	}
	return nil
}
