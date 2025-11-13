package nats

import (
	"fmt"
	"sync"
	"time"

	"github.com/getevo/evo/v2/lib/log"
	"github.com/nats-io/nats.go"
)

var (
	NC   *nats.Conn
	JS   nats.JetStreamContext
	mu   sync.RWMutex
	once sync.Once
)

// NATSConfig holds NATS connection configuration
type NATSConfig struct {
	URL             string        `yaml:"URL"`
	ClusterID       string        `yaml:"CLUSTER_ID"`
	MaxReconnects   int           `yaml:"MAX_RECONNECTS"`
	ReconnectWait   time.Duration `yaml:"RECONNECT_WAIT"`
	PingInterval    time.Duration `yaml:"PING_INTERVAL"`
	MaxPingsOut     int           `yaml:"MAX_PINGS_OUT"`
	AllowReconnect  bool          `yaml:"ALLOW_RECONNECT"`
	DrainTimeout    time.Duration `yaml:"DRAIN_TIMEOUT"`
}

// Connect establishes a fault-tolerant connection to NATS
func Connect(config NATSConfig) error {
	var err error

	// Connection options with fault tolerance
	opts := []nats.Option{
		nats.Name("homa-service"),
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
		nats.PingInterval(config.PingInterval),
		nats.MaxPingsOutstanding(config.MaxPingsOut),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Warning("NATS disconnected: %v", err)
			} else {
				log.Warning("NATS disconnected")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info("NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			if nc.LastError() != nil {
				log.Error("NATS connection closed: %v", nc.LastError())
			} else {
				log.Info("NATS connection closed")
			}
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			if sub != nil {
				log.Error("NATS error on subscription %s: %v", sub.Subject, err)
			} else {
				log.Error("NATS async error: %v", err)
			}
		}),
	}

	// Allow reconnect based on config
	if !config.AllowReconnect {
		opts = append(opts, nats.NoReconnect())
	}

	// Attempt to connect
	mu.Lock()
	NC, err = nats.Connect(config.URL, opts...)
	mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to connect to NATS at %s: %w", config.URL, err)
	}

	log.Info("Connected to NATS at %s", NC.ConnectedUrl())
	log.Info("NATS server info: %s (version: %s)", NC.ConnectedServerName(), NC.ConnectedServerVersion())

	// Initialize JetStream for persistence and advanced features
	mu.Lock()
	JS, err = NC.JetStream()
	mu.Unlock()

	if err != nil {
		log.Warning("JetStream not available: %v", err)
		log.Warning("Continuing without JetStream - advanced features may be limited")
	} else {
		log.Info("JetStream initialized successfully")
	}

	return nil
}

// GetConnection returns the NATS connection
func GetConnection() *nats.Conn {
	mu.RLock()
	defer mu.RUnlock()
	return NC
}

// GetJetStream returns the JetStream context
func GetJetStream() nats.JetStreamContext {
	mu.RLock()
	defer mu.RUnlock()
	return JS
}

// IsConnected checks if NATS is connected
func IsConnected() bool {
	mu.RLock()
	defer mu.RUnlock()
	return NC != nil && NC.IsConnected()
}

// Close gracefully closes the NATS connection with drain
func Close(drainTimeout time.Duration) error {
	mu.Lock()
	defer mu.Unlock()

	if NC == nil {
		return nil
	}

	// Drain for graceful shutdown
	if err := NC.Drain(); err != nil {
		log.Warning("Error draining NATS connection: %v", err)
		NC.Close()
		return err
	}

	// Wait for drain to complete or timeout
	select {
	case <-time.After(drainTimeout):
		log.Warning("Drain timeout exceeded, forcing close")
		NC.Close()
	case <-time.After(100 * time.Millisecond):
		// Small delay to allow drain to complete
	}

	log.Info("NATS connection closed gracefully")
	return nil
}

// Publish publishes a message to a subject
func Publish(subject string, data []byte) error {
	conn := GetConnection()
	if conn == nil || !conn.IsConnected() {
		return fmt.Errorf("NATS not connected")
	}

	return conn.Publish(subject, data)
}

// Subscribe creates a subscription to a subject
func Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	conn := GetConnection()
	if conn == nil || !conn.IsConnected() {
		return nil, fmt.Errorf("NATS not connected")
	}

	return conn.Subscribe(subject, handler)
}

// QueueSubscribe creates a queue subscription to a subject
func QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	conn := GetConnection()
	if conn == nil || !conn.IsConnected() {
		return nil, fmt.Errorf("NATS not connected")
	}

	return conn.QueueSubscribe(subject, queue, handler)
}

// Request sends a request and waits for a response
func Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	conn := GetConnection()
	if conn == nil || !conn.IsConnected() {
		return nil, fmt.Errorf("NATS not connected")
	}

	return conn.Request(subject, data, timeout)
}
