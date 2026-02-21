package redis

import (
	"collaborative-markdown-editor/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	Client *redis.Client
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{Client: client}
}

func NewRedisClient() (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:         config.AppConfig.RedisAddress,
        PoolSize:     config.AppConfig.RedisPollSize,              // Connection pooling 
        MinIdleConns: 3,
        DialTimeout:  5 * time.Second,
    })

    // Use a short timeout for the initial ping
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("redis connection failed: %w", err)
    }

    return client, nil
}

// Stores any object as JSON
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c.Client == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.Client.Set(ctx, key, data, ttl).Err()
}

// set a key's value only if the key does not already exist. (return false -> key already exist)
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	if c.Client == nil {
		return true, nil
	}

	return c.Client.SetNX(ctx, key, value, ttl).Result()
}

// Retrieves JSON and unmarshals it to variable
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	if c.Client == nil {
		return false, nil
	}

	val, err := c.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *Cache) Invalidate(ctx context.Context, key string) error {
	if c.Client == nil {
		return nil
	}
	return c.Client.Del(ctx, key).Err()
}

func (c *Cache) GetVersion(ctx context.Context, namespace string) int64 {
    if c.Client == nil {
        return 0
    }
    // Namespace would be something like "user:1:docs:version"
    val, _ := c.Client.Get(ctx, namespace).Int64()
    return val
}

func (c *Cache) IncrementVersion(ctx context.Context, namespace string) {
    if c.Client == nil {
		return
	}
	// same like .Set with no ttl (stay forever)
    c.Client.Incr(ctx, namespace)
	// set expire to 1 week
    c.Client.Expire(ctx, namespace, 7*24*time.Hour)
}