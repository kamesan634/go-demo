package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-demo/chat/internal/dto/response"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// InMemoryRateLimiter implements rate limiting using in-memory token bucket
type InMemoryRateLimiter struct {
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

// NewInMemoryRateLimiter creates a new in-memory rate limiter
func NewInMemoryRateLimiter(r rate.Limit, burst int) *InMemoryRateLimiter {
	return &InMemoryRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst,
	}
}

// Allow checks if request is allowed
func (l *InMemoryRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	limiter, exists := l.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(l.rate, l.burst)
		l.limiters[key] = limiter
	}
	return limiter.Allow(), nil
}

// RedisRateLimiter implements rate limiting using Redis
type RedisRateLimiter struct {
	client   *redis.Client
	requests int
	window   time.Duration
}

// NewRedisRateLimiter creates a new Redis rate limiter
func NewRedisRateLimiter(client *redis.Client, requests int, window time.Duration) *RedisRateLimiter {
	return &RedisRateLimiter{
		client:   client,
		requests: requests,
		window:   window,
	}
}

// Allow checks if request is allowed using Redis sliding window
func (l *RedisRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	pipe := l.client.Pipeline()

	now := time.Now().UnixNano()
	windowStart := now - l.window.Nanoseconds()

	// Remove old entries
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))

	// Add current request
	pipe.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: now,
	})

	// Count requests in window
	countCmd := pipe.ZCard(ctx, key)

	// Set expiration
	pipe.Expire(ctx, key, l.window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count, err := countCmd.Result()
	if err != nil {
		return false, err
	}

	return count <= int64(l.requests), nil
}

// RateLimitConfig represents rate limit configuration
type RateLimitConfig struct {
	Requests int           // Number of requests allowed
	Window   time.Duration // Time window
	KeyFunc  func(*gin.Context) string
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Requests: 100,
		Window:   time.Minute,
		KeyFunc: func(c *gin.Context) string {
			// Use user ID if authenticated, otherwise use IP
			if userID := GetUserID(c); userID != "" {
				return "ratelimit:user:" + userID
			}
			return "ratelimit:ip:" + c.ClientIP()
		},
	}
}

// RateLimit creates a rate limiting middleware
func RateLimit(limiter RateLimiter) gin.HandlerFunc {
	return RateLimitWithConfig(limiter, DefaultRateLimitConfig())
}

// RateLimitWithConfig creates a rate limiting middleware with custom configuration
func RateLimitWithConfig(limiter RateLimiter, config *RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := config.KeyFunc(c)

		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			// On error, allow the request but log the error
			c.Next()
			return
		}

		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(config.Window.Seconds())))
			response.ErrorWithStatus(c, http.StatusTooManyRequests, "請求過於頻繁，請稍後再試")
			c.Abort()
			return
		}

		c.Next()
	}
}

// APIRateLimit creates a rate limit for general API endpoints
func APIRateLimit(client *redis.Client) gin.HandlerFunc {
	limiter := NewRedisRateLimiter(client, 100, time.Minute)
	return RateLimit(limiter)
}

// AuthRateLimit creates a stricter rate limit for auth endpoints
func AuthRateLimit(client *redis.Client) gin.HandlerFunc {
	limiter := NewRedisRateLimiter(client, 10, time.Minute)
	config := &RateLimitConfig{
		Requests: 10,
		Window:   time.Minute,
		KeyFunc: func(c *gin.Context) string {
			return "ratelimit:auth:" + c.ClientIP()
		},
	}
	return RateLimitWithConfig(limiter, config)
}

// MessageRateLimit creates a rate limit for message sending
func MessageRateLimit(client *redis.Client) gin.HandlerFunc {
	limiter := NewRedisRateLimiter(client, 60, time.Minute)
	config := &RateLimitConfig{
		Requests: 60,
		Window:   time.Minute,
		KeyFunc: func(c *gin.Context) string {
			if userID := GetUserID(c); userID != "" {
				return "ratelimit:message:" + userID
			}
			return "ratelimit:message:" + c.ClientIP()
		},
	}
	return RateLimitWithConfig(limiter, config)
}
