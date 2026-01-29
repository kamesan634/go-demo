package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/go-demo/chat/internal/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewRedis(cfg *config.RedisConfig, logger *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("Connected to Redis",
		zap.String("addr", cfg.GetAddr()),
		zap.Int("db", cfg.DB),
	)

	return client, nil
}

// Close closes the Redis connection
func Close(client *redis.Client, logger *zap.Logger) {
	if err := client.Close(); err != nil {
		logger.Error("Error closing Redis connection", zap.Error(err))
	} else {
		logger.Info("Redis connection closed")
	}
}

// Cache provides common caching operations
type Cache struct {
	client *redis.Client
	logger *zap.Logger
}

func NewCache(client *redis.Client, logger *zap.Logger) *Cache {
	return &Cache{
		client: client,
		logger: logger,
	}
}

// Set stores a value with expiration
func (c *Cache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Delete removes a key
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	return result > 0, err
}

// SetNX sets a value only if it doesn't exist (for distributed locks)
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, expiration).Result()
}

// Increment increments a counter
func (c *Cache) Increment(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Expire sets expiration on a key
func (c *Cache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// Keys for chat system
const (
	KeyUserOnline     = "user:online:%s"      // user:online:{userID}
	KeyRoomMembers    = "room:members:%s"     // room:members:{roomID}
	KeyUserRooms      = "user:rooms:%s"       // user:rooms:{userID}
	KeyRateLimitUser  = "ratelimit:user:%s"   // ratelimit:user:{userID}
	KeyRateLimitIP    = "ratelimit:ip:%s"     // ratelimit:ip:{ip}
	KeyRefreshToken   = "refresh_token:%s"    // refresh_token:{tokenID}
	KeyBlockedTokens  = "blocked_tokens"      // Set of blocked JWT token IDs
)
