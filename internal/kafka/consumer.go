package kafka

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/rs/zerolog/log"

	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/event"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type KafkaConsumer struct {
	consumer     *kafka.Consumer
	eventService *event.Service
	workers      map[int32]chan *kafka.Message
	isRunning    atomic.Bool
	mu           sync.Mutex
	wg           sync.WaitGroup
}

func NewKafkaConsumer(eventService *event.Service, groupID string, topics []string) (*KafkaConsumer, error) {
	if config.AppConfig.KafkaBootstrapServers == "" {
		return nil, errors.New("Kafka is not configured")
	}

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
	kc := &KafkaConsumer{
		consumer:     c,
		eventService: eventService,
		workers:      make(map[int32]chan *kafka.Message),
	}

	c.SubscribeTopics(topics, func(c *kafka.Consumer, e kafka.Event) error {
		// rebalance
		switch ev := e.(type) {
		case kafka.AssignedPartitions:
			log.Debug().Msgf("Assigned: %v", ev.Partitions)
			c.Assign(ev.Partitions)

		case kafka.RevokedPartitions:
			log.Warn().Msgf("Revoked: %v", ev.Partitions)

			kc.mu.Lock()
			for _, p := range ev.Partitions {
				if ch, ok := kc.workers[p.Partition]; ok {
					delete(kc.workers, p.Partition)
					close(ch)
				}
			}
			kc.mu.Unlock()

			c.Unassign()
		}
		return nil
	})
	
	kc.isRunning.Store(true)

	return kc, nil
}

func (kc *KafkaConsumer) Start() error {
	log.Info().Msg("Kafka Consumer is running")
	for kc.isRunning.Load() {
		ev := kc.consumer.Poll(100) // 100 ms
		if ev == nil {
			continue
		}
		switch e := ev.(type) {
		case *kafka.Message:
			partition := e.TopicPartition.Partition

			kc.mu.Lock()
			if kc.workers[partition] == nil {
				kc.workers[partition] = make(chan *kafka.Message, 100)

				kc.wg.Add(1)
				go func() {
					defer kc.wg.Done()
					kc.startPartitionWorker(partition, kc.workers[partition])
				}()
			}
			channel := kc.workers[partition]
			kc.mu.Unlock()

			select {
			case channel <- e:
			default:
				log.Warn().Msg("Partition channel full, slowing down")
				channel <- e
			}

		case kafka.Error:
			// ERRORS: Handle connection issues
			log.Error().Err(e).Msgf("Consumer error %v", e.IsFatal())
			if e.IsFatal() {
				return kc.Close()
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
		err := kc.processMessage(context.Background(), msg)

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
	kc.isRunning.Store(false)

	kc.mu.Lock()
	for _, ch := range kc.workers {
		close(ch)
	}
	kc.mu.Unlock()
	kc.wg.Wait()

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
		return errors.New("Unknown topic")
	}
}
