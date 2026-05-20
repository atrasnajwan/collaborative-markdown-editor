package kafka

import (
	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/worker"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/rs/zerolog/log"
)

type KafkaProducer struct {
	producer *kafka.Producer
	wp       *worker.WorkerPool
}

func NewKafkaProducer(wp *worker.WorkerPool) (*KafkaProducer, error) {
	if config.AppConfig.KafkaBootstrapServers == "" {
		return nil, errors.New("Kafka is not configured")
	}
	
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":     config.AppConfig.KafkaBootstrapServers,
		"enable.idempotence":    true,
		"message.max.bytes":     10 * 1024 * 1024, // 10MB max
		"request.required.acks": "all",
	})

	if err != nil {
		return nil, err
	}

	// Delivery report handler for produced messages
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Error().Err(ev.TopicPartition.Error).Msgf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					log.Debug().Msgf("Delivered message to %v\n", ev.TopicPartition)
				}
			}
		}
	}()
	
	return &KafkaProducer{
		producer: p,
		wp:       wp,
	}, nil
}

func (kp *KafkaProducer) SendMessage(topic string, key string, message interface{}) {
	payload, err := json.Marshal(message)
	if err != nil {
		log.Error().Err(err).Msg("Failed to decode json")
		return
	}
	kp.wp.Submit(func(bgCtx context.Context) error {
		err := kp.producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{
				Topic:     &topic,
				Partition: kafka.PartitionAny,
			},
			Key:   []byte(key),
			Value: payload,
		}, nil)

		if err != nil {
			log.Error().Err(err).Msg("Failed to send event")
		}
		return err
	})
}

func (kp *KafkaProducer) Close() {
	kp.producer.Flush(15 * int(time.Second))
	kp.producer.Close()
}
