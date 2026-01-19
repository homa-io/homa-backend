package redis

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/getevo/evo/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/iesreza/homa-backend/lib/response"
)

// RateLimitConfig holds the configuration for a rate limit rule
type RateLimitConfig struct {
	MaxRequests int
	Window      time.Duration
	Enabled     bool
}

// DefaultRateLimitConfig returns a default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxRequests: 60,              // 60 requests
		Window:      1 * time.Minute, // per minute
		Enabled:     true,
	}
}

// rateLimitCache stores rate limit configurations in memory
var (
	rateLimitCache sync.Map
	cacheMutex     sync.RWMutex
)

// SetRateLimitConfig sets a rate limit configuration for a key
func SetRateLimitConfig(key string, config RateLimitConfig) {
	rateLimitCache.Store(key, config)
}

// GetRateLimitConfig gets a rate limit configuration for a key
func GetRateLimitConfig(key string) RateLimitConfig {
	if cached, ok := rateLimitCache.Load(key); ok {
		return cached.(RateLimitConfig)
	}
	return DefaultRateLimitConfig()
}

// ClearRateLimitCache clears all cached rate limit configurations
func ClearRateLimitCache() {
	rateLimitCache = sync.Map{}
	log.Println("Rate limit cache cleared")
}

// RateLimitMiddleware creates a rate limiting middleware for a specific key
func RateLimitMiddleware(key string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if Redis is not available
		if !IsAvailable() {
			return c.Next()
		}

		config := GetRateLimitConfig(key)

		// Skip if rate limiting is disabled for this endpoint
		if !config.Enabled {
			return c.Next()
		}

		// Get client identifier (IP address)
		clientIP := c.IP()
		if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = forwarded
		}

		// Create Redis key for this client and endpoint
		redisKey := fmt.Sprintf("rate_limit:%s:%s", key, clientIP)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Increment the counter
		count, err := Client.Incr(ctx, redisKey).Result()
		if err != nil {
			log.Printf("Redis rate limit error: %v", err)
			return c.Next() // Allow request on Redis error
		}

		// Set expiry on first request
		if count == 1 {
			Client.Expire(ctx, redisKey, config.Window)
		}

		// Get TTL for retry-after header
		ttl, _ := Client.TTL(ctx, redisKey).Result()

		// Add rate limit headers
		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.MaxRequests))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, config.MaxRequests-int(count))))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(ttl).Unix()))

		// Check if rate limit exceeded
		if int(count) > config.MaxRequests {
			c.Set("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Too many requests",
				"retry_after": int(ttl.Seconds()),
			})
		}

		return c.Next()
	}
}

// RateLimitByIP creates a generic IP-based rate limiter
func RateLimitByIP(maxRequests int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip if Redis is not available
		if !IsAvailable() {
			return c.Next()
		}

		// Get client identifier
		clientIP := c.IP()
		if forwarded := c.Get("X-Forwarded-For"); forwarded != "" {
			clientIP = forwarded
		}

		redisKey := fmt.Sprintf("rate_limit:ip:%s", clientIP)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		count, err := Client.Incr(ctx, redisKey).Result()
		if err != nil {
			return c.Next()
		}

		if count == 1 {
			Client.Expire(ctx, redisKey, window)
		}

		ttl, _ := Client.TTL(ctx, redisKey).Result()

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, maxRequests-int(count))))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(ttl).Unix()))

		if int(count) > maxRequests {
			c.Set("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "Too many requests",
				"retry_after": int(ttl.Seconds()),
			})
		}

		return c.Next()
	}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// EvoRateLimitMiddleware creates an evo-compatible rate limiting middleware
func EvoRateLimitMiddleware(key string) func(*evo.Request) error {
	return func(req *evo.Request) error {
		// Skip if Redis is not available
		if !IsAvailable() {
			return req.Next()
		}

		config := GetRateLimitConfig(key)

		// Skip if rate limiting is disabled for this endpoint
		if !config.Enabled {
			return req.Next()
		}

		// Get client identifier (IP address)
		clientIP := req.IP()
		if forwarded := req.Header("X-Forwarded-For"); forwarded != "" {
			clientIP = forwarded
		}

		// Create Redis key for this client and endpoint
		redisKey := fmt.Sprintf("rate_limit:%s:%s", key, clientIP)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Increment the counter
		count, err := Client.Incr(ctx, redisKey).Result()
		if err != nil {
			log.Printf("Redis rate limit error: %v", err)
			return req.Next() // Allow request on Redis error
		}

		// Set expiry on first request
		if count == 1 {
			Client.Expire(ctx, redisKey, config.Window)
		}

		// Check if rate limit exceeded
		if int(count) > config.MaxRequests {
			return response.NewError("too_many_requests", "Too many requests. Please try again later.", 429)
		}

		return req.Next()
	}
}
