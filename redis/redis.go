package redis

import (
	"collaborative-markdown-editor/internal/config"
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()
var RedisClient *redis.Client

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr: config.AppConfig.RedisAddress,
	})
	_, err := RedisClient.Ping(Ctx).Result()
	if err != nil {
		log.Println("Redis not available. Running without Redis.")
		RedisClient = nil
		return
	}

	log.Println("Redis connected successfully.")
}
