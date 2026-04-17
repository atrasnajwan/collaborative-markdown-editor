package kafka

import (
	"context"
	"github.com/rs/zerolog/log"

	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/event"
	"collaborative-markdown-editor/internal/worker"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaConsumer struct {
	consumer     *kafka.Consumer
	wp           *worker.WorkerPool
	eventService event.Service
	workers      map[int32]chan *kafka.Message
	isRunning    bool
}



func NewKafkaConsumer(workerPool *worker.WorkerPool, eventService event.Service, groupID string, topics []string) (*KafkaConsumer, error) {
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
		return kc.eventService.ProcessDocumentEvent(ctx, message)
	default:
		log.Debug().Msgf("Unknown topic %s", *message.TopicPartition.Topic)
	}
	return nil
}
