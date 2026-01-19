package redis

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/getevo/evo/v2/lib/settings"
	"github.com/redis/go-redis/v9"
)

var (
	// Client is the universal Redis client that works with both single nodes and clusters
	Client redis.UniversalClient
	ctx    = context.Background()
)

// RedisConfig holds the Redis configuration
type RedisConfig struct {
	Addresses    []string      `json:"addresses"`
	Password     string        `json:"password"`
	DB           int           `json:"db"`
	MaxRetries   int           `json:"max_retries"`
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	PoolSize     int           `json:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns"`
	// Cluster-specific settings
	RouteByLatency bool `json:"route_by_latency"`
	RouteRandomly  bool `json:"route_randomly"`
	// Sentinel-specific settings (MasterName triggers sentinel mode)
	MasterName       string `json:"master_name"`
	SentinelPassword string `json:"sentinel_password"`
}

// Initialize creates a new Redis universal client connection
// Supports both single node and cluster configurations via config.yml
//
// Example config.yml for single node:
//
//	REDIS:
//	  ADDRESS: "localhost:6379"
//	  PASSWORD: ""
//	  DB: 0
//
// Example config.yml for cluster (multiple nodes):
//
//	REDIS:
//	  ADDRESSES: "redis1:6379,redis2:6379,redis3:6379"
//	  PASSWORD: ""
//
// Example config.yml for sentinel (high availability):
//
//	REDIS:
//	  ADDRESSES: "sentinel1:26379,sentinel2:26379,sentinel3:26379"
//	  MASTER_NAME: "mymaster"
//	  PASSWORD: ""
//	  SENTINEL_PASSWORD: ""
func Initialize() error {
	config := loadConfig()

	// Skip initialization if no addresses configured
	if len(config.Addresses) == 0 {
		log.Println("Redis not configured. Rate limiting will be disabled.")
		return nil
	}

	// Create universal client options
	opts := &redis.UniversalOptions{
		Addrs:        config.Addresses,
		Password:     config.Password,
		DB:           config.DB,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		PoolSize:     config.PoolSize,
		MinIdleConns: config.MinIdleConns,
		// Cluster options
		RouteByLatency: config.RouteByLatency,
		RouteRandomly:  config.RouteRandomly,
		// Sentinel options (MasterName triggers sentinel/failover mode)
		MasterName:       config.MasterName,
		SentinelPassword: config.SentinelPassword,
	}

	// NewUniversalClient returns:
	// - ClusterClient when len(Addrs) > 1 and no MasterName
	// - FailoverClient when MasterName is set (Sentinel mode)
	// - Simple Client when len(Addrs) == 1 and no MasterName
	Client = redis.NewUniversalClient(opts)

	// Test connection
	testCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(testCtx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v. Rate limiting will be disabled.", err)
		Client = nil
		return nil // Don't fail startup if Redis is unavailable
	}

	// Log connection mode
	switch len(config.Addresses) {
	case 0:
		if config.MasterName != "" {
			log.Printf("Redis Sentinel connected (master: %s)", config.MasterName)
		}
	case 1:
		log.Printf("Redis connected (single node: %s)", config.Addresses[0])
	default:
		log.Printf("Redis Cluster connected (%d nodes)", len(config.Addresses))
	}

	return nil
}

// loadConfig reads Redis configuration from settings
func loadConfig() RedisConfig {
	config := RedisConfig{
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	}

	// Try to get addresses as array first (for cluster/multiple nodes)
	// REDIS.ADDRESSES: ["redis1:6379", "redis2:6379"]
	addressesRaw := settings.Get("REDIS.ADDRESSES")
	if addressesRaw.String() != "" {
		// Try parsing as comma-separated string or get the raw value
		addrStr := addressesRaw.String()
		if strings.Contains(addrStr, ",") {
			// Comma-separated format: "redis1:6379,redis2:6379"
			addrs := strings.Split(addrStr, ",")
			for _, addr := range addrs {
				addr = strings.TrimSpace(addr)
				if addr != "" {
					config.Addresses = append(config.Addresses, addr)
				}
			}
		} else if addrStr != "" && addrStr != "[]" {
			// Single address in ADDRESSES field
			config.Addresses = append(config.Addresses, strings.TrimSpace(addrStr))
		}
	}

	// Fall back to single ADDRESS (legacy/simple config)
	// REDIS.ADDRESS: "localhost:6379"
	if len(config.Addresses) == 0 {
		singleAddr := settings.Get("REDIS.ADDRESS").String()
		if singleAddr == "" {
			// Also try legacy REDIS_URL format
			singleAddr = settings.Get("REDIS_URL").String()
		}
		if singleAddr != "" {
			// Support comma-separated in ADDRESS field too
			if strings.Contains(singleAddr, ",") {
				addrs := strings.Split(singleAddr, ",")
				for _, addr := range addrs {
					addr = strings.TrimSpace(addr)
					if addr != "" {
						config.Addresses = append(config.Addresses, addr)
					}
				}
			} else {
				config.Addresses = append(config.Addresses, singleAddr)
			}
		}
	}

	// Password
	config.Password = settings.Get("REDIS.PASSWORD").String()
	if config.Password == "" {
		config.Password = settings.Get("REDIS_PASSWORD").String()
	}

	// Database (only used for non-cluster mode)
	config.DB = settings.Get("REDIS.DB").Int()
	if config.DB == 0 {
		config.DB = settings.Get("REDIS_DB").Int()
	}

	// Optional: connection pool settings
	if poolSize := settings.Get("REDIS.POOL_SIZE").Int(); poolSize > 0 {
		config.PoolSize = poolSize
	}
	if minIdle := settings.Get("REDIS.MIN_IDLE_CONNS").Int(); minIdle > 0 {
		config.MinIdleConns = minIdle
	}
	if maxRetries := settings.Get("REDIS.MAX_RETRIES").Int(); maxRetries > 0 {
		config.MaxRetries = maxRetries
	}

	// Cluster-specific settings
	config.RouteByLatency = settings.Get("REDIS.ROUTE_BY_LATENCY").Bool()
	config.RouteRandomly = settings.Get("REDIS.ROUTE_RANDOMLY").Bool()

	// Sentinel settings (when MasterName is set, Addrs are treated as sentinel addresses)
	config.MasterName = settings.Get("REDIS.MASTER_NAME").String()
	config.SentinelPassword = settings.Get("REDIS.SENTINEL_PASSWORD").String()

	return config
}

// IsAvailable returns true if Redis client is connected
func IsAvailable() bool {
	if Client == nil {
		return false
	}
	return Client.Ping(ctx).Err() == nil
}

// Close gracefully closes the Redis connection
func Close() error {
	if Client != nil {
		return Client.Close()
	}
	return nil
}
