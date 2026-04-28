package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisClient struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedisClient(addr string, password string, db int, logger *zap.Logger) (*RedisClient, error) {
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
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Successfully connected to Redis")
	return &RedisClient{
		client: client,
		logger: logger,
	}, nil
}

func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

// GetWithRefresh 获取缓存并刷新过期时间
// 当缓存命中时，自动将过期时间延长为新的 expiration
func (r *RedisClient) GetWithRefresh(ctx context.Context, key string, expiration time.Duration) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	// 使用独立的 context，避免父 context 取消影响
	go func() {
		refreshCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := r.client.Expire(refreshCtx, key, expiration).Err(); err != nil {
			r.logger.Warn("Failed to refresh cache expiration", zap.Error(err), zap.String("key", key))
		}
	}()

	return result, nil
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

// 用于敏感词集合的方法
func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return r.client.SAdd(ctx, key, members...).Err()
}

func (r *RedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// ========== Redis Stream 操作（用于异步 LLM 审核 MQ） ==========

const (
	// LLMReviewStream Redis Stream 名称
	LLMReviewStream = "llm_review_stream"
	// LLMReviewGroup 消费组名称
	LLMReviewGroup = "llm_review_workers"
)

// XAdd 向 Redis Stream 添加消息
func (r *RedisClient) XAdd(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
		MaxLen: 10000, // 保留最近 10000 条消息，防止 Stream 无限增长
	}).Result()
}

// XReadGroup 从消费组读取消息（阻塞模式）
func (r *RedisClient) XReadGroup(ctx context.Context, stream, group, consumer string, count int64) ([]redis.XStream, error) {
	return r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    5 * time.Second, // 5 秒阻塞等待，避免空轮询
	}).Result()
}

// XAck 确认消息已处理
func (r *RedisClient) XAck(ctx context.Context, stream, group string, ids ...string) error {
	return r.client.XAck(ctx, stream, group, ids...).Err()
}

// XGroupCreate 创建消费组（如果不存在）
func (r *RedisClient) XGroupCreate(ctx context.Context, stream, group, start string) error {
	err := r.client.XGroupCreateMkStream(ctx, stream, group, start).Err()
	if err != nil && strings.Contains(err.Error(), "BUSYGROUP") {
		// 消费组已存在，不是错误
		return nil
	}
	return err
}
