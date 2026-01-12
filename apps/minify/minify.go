package minify

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getevo/evo/v2/lib/log"
	"github.com/gofiber/fiber/v2"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/js"
)

// Cache entry for minified content
type cacheEntry struct {
	content       []byte
	version       string
	sourceModTime time.Time
	minifiedAt    time.Time
}

// JSMinifier handles JavaScript minification with caching
type JSMinifier struct {
	cache    map[string]*cacheEntry
	mutex    sync.RWMutex
	minifier *minify.M
}

var defaultMinifier *JSMinifier

func init() {
	defaultMinifier = NewJSMinifier()
}

// NewJSMinifier creates a new JS minifier instance
func NewJSMinifier() *JSMinifier {
	m := minify.New()
	m.AddFunc("application/javascript", js.Minify)

	return &JSMinifier{
		cache:    make(map[string]*cacheEntry),
		minifier: m,
	}
}

// getCacheKey generates a cache key from file path and version
func getCacheKey(filePath, version string) string {
	return filePath + ":" + version
}

// Controller holds the minify handlers
type Controller struct {
	basePath string
}

// NewController creates a new minify controller
func NewController(basePath string) *Controller {
	return &Controller{basePath: basePath}
}

// ServeMinifiedJS handles requests for minified JS files (Fiber handler)
func (ctrl *Controller) ServeMinifiedJS(c *fiber.Ctx) error {
	return defaultMinifier.ServeMinified(c, ctrl.basePath)
}

// ServeMinified serves a minified version of the requested JS file
func (jm *JSMinifier) ServeMinified(c *fiber.Ctx, basePath string) error {
	// Get the requested file path (e.g., "homa-chat.min.js")
	requestedFile := c.Params("*")

	// Only handle .js files
	if filepath.Ext(requestedFile) != ".js" {
		return c.Status(404).SendString("Not found")
	}

	// Check if it's a .min.js request
	baseName := requestedFile
	if len(requestedFile) > 7 && requestedFile[len(requestedFile)-7:] == ".min.js" {
		// Remove .min.js and add .js to get source file
		baseName = requestedFile[:len(requestedFile)-7] + ".js"
	} else {
		// Not a minified request
		return c.Status(404).SendString("Not a minified file request")
	}

	// Get version from query string
	version := c.Query("v", "")

	// Build source file path
	sourceFile := filepath.Join(basePath, baseName)

	// Check if source file exists
	fileInfo, err := os.Stat(sourceFile)
	if err != nil {
		log.Warning("Minify: Source file not found: %s", sourceFile)
		return c.Status(404).SendString("Source file not found")
	}

	// Generate cache key
	cacheKey := getCacheKey(sourceFile, version)

	// Check cache
	jm.mutex.RLock()
	entry, exists := jm.cache[cacheKey]
	jm.mutex.RUnlock()

	// Return cached version if valid
	if exists && entry.sourceModTime.Equal(fileInfo.ModTime()) {
		c.Set("Content-Type", "application/javascript; charset=utf-8")
		c.Set("Cache-Control", "public, max-age=31536000") // 1 year cache
		c.Set("X-Minified", "true")
		c.Set("X-Minified-At", entry.minifiedAt.Format(time.RFC3339))
		return c.Send(entry.content)
	}

	// Read source file
	source, err := os.ReadFile(sourceFile)
	if err != nil {
		log.Error("Minify: Failed to read source file: %v", err)
		return c.Status(500).SendString("Failed to read source file")
	}

	// Minify the content
	minified, err := jm.minifier.Bytes("application/javascript", source)
	if err != nil {
		log.Error("Minify: Failed to minify file: %v", err)
		// Fall back to serving original content
		c.Set("Content-Type", "application/javascript; charset=utf-8")
		c.Set("X-Minified", "false")
		c.Set("X-Minify-Error", err.Error())
		return c.Send(source)
	}

	// Store in cache
	now := time.Now()
	jm.mutex.Lock()
	jm.cache[cacheKey] = &cacheEntry{
		content:       minified,
		version:       version,
		sourceModTime: fileInfo.ModTime(),
		minifiedAt:    now,
	}
	jm.mutex.Unlock()

	log.Info("Minify: Minified %s (v=%s) - Original: %d bytes, Minified: %d bytes (%.1f%% reduction)",
		baseName, version, len(source), len(minified),
		(1-float64(len(minified))/float64(len(source)))*100)

	// Serve the minified content
	c.Set("Content-Type", "application/javascript; charset=utf-8")
	c.Set("Cache-Control", "public, max-age=31536000") // 1 year cache
	c.Set("X-Minified", "true")
	c.Set("X-Minified-At", now.Format(time.RFC3339))
	c.Set("X-Original-Size", fmt.Sprintf("%d", len(source)))
	c.Set("X-Minified-Size", fmt.Sprintf("%d", len(minified)))

	return c.Send(minified)
}

// ClearCache clears all cached minified files
func (jm *JSMinifier) ClearCache() {
	jm.mutex.Lock()
	jm.cache = make(map[string]*cacheEntry)
	jm.mutex.Unlock()
	log.Info("Minify: Cache cleared")
}

// ClearCacheForFile clears cache for a specific file
func (jm *JSMinifier) ClearCacheForFile(filePath string) {
	jm.mutex.Lock()
	for key := range jm.cache {
		if len(key) >= len(filePath) && key[:len(filePath)] == filePath {
			delete(jm.cache, key)
		}
	}
	jm.mutex.Unlock()
	log.Info("Minify: Cache cleared for %s", filePath)
}

// GetCacheStats returns cache statistics
func (jm *JSMinifier) GetCacheStats() map[string]interface{} {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	totalSize := 0
	entries := make([]map[string]interface{}, 0)

	for key, entry := range jm.cache {
		entries = append(entries, map[string]interface{}{
			"key":        key,
			"version":    entry.version,
			"size":       len(entry.content),
			"minifiedAt": entry.minifiedAt,
		})
		totalSize += len(entry.content)
	}

	return map[string]interface{}{
		"entries":     len(jm.cache),
		"totalSize":   totalSize,
		"cachedFiles": entries,
	}
}

// DefaultMinifier returns the default minifier instance
func DefaultMinifier() *JSMinifier {
	return defaultMinifier
}
