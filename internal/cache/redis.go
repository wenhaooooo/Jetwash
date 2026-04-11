package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisClient struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisClient(addr string, password string, db int, logger *zap.Logger) *RedisClient {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("Failed to connect to Redis", zap.Error(err))
	}

	return &RedisClient{
		client: client,
		logger: logger,
	}
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) Del(ctx context.Context, keys ...string) error {
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

// 用于检测队列的方法
func (r *RedisClient) Enqueue(ctx context.Context, queueName string, task string) error {
	return r.client.LPush(ctx, queueName, task).Err()
}

func (r *RedisClient) Dequeue(ctx context.Context, queueName string) (string, error) {
	result, err := r.client.BRPop(ctx, 0, queueName).Result()
	if err != nil {
		return "", err
	}
	return result[1], nil
}
