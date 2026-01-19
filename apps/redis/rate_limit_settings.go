package redis

import (
	"log"
	"strconv"
	"time"

	"github.com/iesreza/homa-backend/apps/models"
	appnats "github.com/iesreza/homa-backend/apps/nats"
	"github.com/nats-io/nats.go"
)

// RateLimitEndpoint represents a rate-limitable endpoint
type RateLimitEndpoint struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MaxRequests int    `json:"max_requests"`
	WindowSecs  int    `json:"window_seconds"`
	Enabled     bool   `json:"enabled"`
}

// DefaultEndpoints returns the list of endpoints that can be rate limited
var DefaultEndpoints = []RateLimitEndpoint{
	{
		Key:         "client.create_conversation",
		Name:        "Create Conversation",
		Description: "Creating new conversations from widget/API",
		MaxRequests: 10,
		WindowSecs:  60,
		Enabled:     true,
	},
	{
		Key:         "client.send_message",
		Name:        "Send Message",
		Description: "Sending messages in conversations",
		MaxRequests: 30,
		WindowSecs:  60,
		Enabled:     true,
	},
	{
		Key:         "client.get_conversation",
		Name:        "Get Conversation",
		Description: "Fetching conversation details",
		MaxRequests: 60,
		WindowSecs:  60,
		Enabled:     true,
	},
	{
		Key:         "widget.load",
		Name:        "Widget Load",
		Description: "Loading the chat widget script",
		MaxRequests: 100,
		WindowSecs:  60,
		Enabled:     true,
	},
	{
		Key:         "kb.search",
		Name:        "Knowledge Base Search",
		Description: "Searching knowledge base articles",
		MaxRequests: 30,
		WindowSecs:  60,
		Enabled:     true,
	},
}

// LoadRateLimitSettings loads rate limit settings from the database into cache
func LoadRateLimitSettings() {
	for _, endpoint := range DefaultEndpoints {
		config := RateLimitConfig{
			MaxRequests: endpoint.MaxRequests,
			Window:      time.Duration(endpoint.WindowSecs) * time.Second,
			Enabled:     endpoint.Enabled,
		}

		// Try to load from database settings using the model's function
		// Load max requests
		if val := models.GetSettingValue("rate_limit."+endpoint.Key+".max_requests", ""); val != "" {
			if intVal, err := strconv.Atoi(val); err == nil {
				config.MaxRequests = intVal
			}
		}

		// Load window
		if val := models.GetSettingValue("rate_limit."+endpoint.Key+".window_seconds", ""); val != "" {
			if intVal, err := strconv.Atoi(val); err == nil {
				config.Window = time.Duration(intVal) * time.Second
			}
		}

		// Load enabled
		if val := models.GetSettingValue("rate_limit."+endpoint.Key+".enabled", ""); val != "" {
			config.Enabled = val == "true" || val == "1"
		}

		// Store in cache
		SetRateLimitConfig(endpoint.Key, config)
	}

	log.Println("Rate limit settings loaded from database")
}

// SaveRateLimitSetting saves a rate limit setting to the database
func SaveRateLimitSetting(key string, maxRequests int, windowSecs int, enabled bool) error {
	// Save max requests
	if err := models.SetSetting("rate_limit."+key+".max_requests", strconv.Itoa(maxRequests), "number", "rate_limit", "Max Requests"); err != nil {
		return err
	}

	// Save window
	if err := models.SetSetting("rate_limit."+key+".window_seconds", strconv.Itoa(windowSecs), "number", "rate_limit", "Window (seconds)"); err != nil {
		return err
	}

	// Save enabled
	enabledStr := "false"
	if enabled {
		enabledStr = "true"
	}
	if err := models.SetSetting("rate_limit."+key+".enabled", enabledStr, "boolean", "rate_limit", "Enabled"); err != nil {
		return err
	}

	// Update cache immediately
	config := RateLimitConfig{
		MaxRequests: maxRequests,
		Window:      time.Duration(windowSecs) * time.Second,
		Enabled:     enabled,
	}
	SetRateLimitConfig(key, config)

	// Publish NATS message to notify other instances
	appnats.Publish("settings.rate_limit.reload", []byte("reload"))

	return nil
}

// GetRateLimitSettings returns all rate limit settings
func GetRateLimitSettings() []RateLimitEndpoint {
	result := make([]RateLimitEndpoint, len(DefaultEndpoints))
	copy(result, DefaultEndpoints)

	for i, endpoint := range result {
		config := GetRateLimitConfig(endpoint.Key)
		result[i].MaxRequests = config.MaxRequests
		result[i].WindowSecs = int(config.Window.Seconds())
		result[i].Enabled = config.Enabled
	}

	return result
}

// SubscribeToRateLimitReload subscribes to NATS for rate limit cache invalidation
func SubscribeToRateLimitReload() {
	_, err := appnats.Subscribe("settings.rate_limit.reload", func(msg *nats.Msg) {
		log.Println("Received rate limit reload signal, refreshing cache...")
		LoadRateLimitSettings()
	})
	if err != nil {
		log.Printf("Failed to subscribe to rate limit reload: %v", err)
	}
}
