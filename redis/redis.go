package redis

import (
	"collaborative-markdown-editor/internal/config"
	"context"

	"github.com/redis/go-redis/v9"
)

var Ctx = context.Background()
var RedisClient *redis.Client

func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr: config.AppConfig.RedisAddress,
	})
}
