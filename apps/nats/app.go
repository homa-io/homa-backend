package nats

import (
	"github.com/getevo/evo/v2/lib/application"
	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
)

// App represents the NATS application module
type App struct{}

// Register initializes the NATS connection
func (App) Register() error {
	log.Info("Registering NATS app...")
	return nil
}

// Router registers HTTP routes (none for NATS)
func (App) Router() error {
	return nil
}

// WhenReady connects to NATS after application is fully initialized
func (App) WhenReady() error {
	log.Info("Initializing NATS connection...")

	// Load NATS configuration from settings
	reconnectWait, _ := settings.Get("NATS.RECONNECT_WAIT", "2s").Duration()
	pingInterval, _ := settings.Get("NATS.PING_INTERVAL", "20s").Duration()
	drainTimeout, _ := settings.Get("NATS.DRAIN_TIMEOUT", "30s").Duration()

	config := NATSConfig{
		URL:            settings.Get("NATS.URL", "nats://localhost:4222").String(),
		ClusterID:      settings.Get("NATS.CLUSTER_ID", "homa-cluster").String(),
		MaxReconnects:  int(settings.Get("NATS.MAX_RECONNECTS", 60).Int64()),
		ReconnectWait:  reconnectWait,
		PingInterval:   pingInterval,
		MaxPingsOut:    int(settings.Get("NATS.MAX_PINGS_OUT", 2).Int64()),
		AllowReconnect: settings.Get("NATS.ALLOW_RECONNECT", true).Bool(),
		DrainTimeout:   drainTimeout,
	}

	// Connect to NATS
	if err := Connect(config); err != nil {
		log.Error("Failed to connect to NATS: %v", err)
		return err
	}

	log.Info("NATS app ready")
	return nil
}

// Name returns the app name
func (App) Name() string {
	return "nats"
}

// Shutdown gracefully closes the NATS connection
func (App) Shutdown() error {
	log.Info("Shutting down NATS connection...")

	// Load drain timeout from config
	drainTimeout, _ := settings.Get("NATS.DRAIN_TIMEOUT", "30s").Duration()
	return Close(drainTimeout)
}

// GetInstance returns the singleton NATS app instance
func GetInstance() *App {
	return &App{}
}

var _ application.Application = (*App)(nil)
