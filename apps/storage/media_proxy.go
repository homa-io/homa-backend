package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/getevo/evo/v2/lib/log"
	"github.com/getevo/evo/v2/lib/settings"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

var (
	cachePath     string
	cacheEnabled  bool
	cacheDuration time.Duration
	uploadPrefix  string
)

// InitMediaProxy initializes the media proxy
func InitMediaProxy() error {
	cachePath = settings.Get("S3.CACHE_PATH").String()
	if cachePath == "" {
		cachePath = "./cache/media"
	}

	// Parse cache duration from config (e.g., "7d", "24h", "1h")
	cacheDurationStr := settings.Get("S3.CACHE_DURATION").String()
	cacheDuration = parseDuration(cacheDurationStr)
	if cacheDuration == 0 {
		cacheDuration = 7 * 24 * time.Hour // Default to 7 days
	}

	// Get upload prefix from config
	uploadPrefix = settings.Get("S3.UPLOAD_PREFIX").String()
	if uploadPrefix == "" {
		uploadPrefix = "uploads"
	}

	// Create cache directory
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cacheEnabled = true
	log.Notice("Media proxy initialized: cache=%s, duration=%v, prefix=%s", cachePath, cacheDuration, uploadPrefix)
	return nil
}

// parseDuration parses duration strings like "7d", "24h", "1h", "30m"
func parseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}

	s = strings.TrimSpace(s)

	// Check for day suffix
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0
		}
		return time.Duration(days) * 24 * time.Hour
	}

	// Try standard Go duration parsing for hours, minutes, etc.
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0
	}
	return d
}

// GetUploadPrefix returns the configured upload prefix
func GetUploadPrefix() string {
	if uploadPrefix == "" {
		return "uploads"
	}
	return uploadPrefix
}

// GetCacheDuration returns the configured cache duration
func GetCacheDuration() time.Duration {
	if cacheDuration == 0 {
		return 7 * 24 * time.Hour
	}
	return cacheDuration
}

// MediaProxyHandler handles media proxy requests
// URL format: /media/{path}?fmt=webp&size=256x-
func MediaProxyHandler(c *fiber.Ctx) error {
	if !IsEnabled() {
		return c.Status(503).JSON(fiber.Map{
			"error": "S3 storage not enabled",
		})
	}

	// Get path from URL
	path := c.Params("*")
	if path == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Path is required",
		})
	}

	// Parse query parameters
	format := c.Query("fmt", "")      // webp, jpg, png
	sizeStr := c.Query("size", "")    // 256x-, -x256, 256x256
	quality := c.Query("q", "85")     // Quality 1-100

	qualityInt, err := strconv.Atoi(quality)
	if err != nil || qualityInt < 1 || qualityInt > 100 {
		qualityInt = 85
	}

	// Check if this is a video request with Range header
	rangeHeader := c.Get("Range")
	isVideo := isVideoPath(path)

	// For video with Range request, handle streaming
	if isVideo && rangeHeader != "" {
		return handleVideoRangeRequest(c, path, rangeHeader)
	}

	// Generate cache key
	cacheKey := generateCacheKey(path, format, sizeStr, qualityInt)

	// Check cache first
	if cacheEnabled {
		cacheFile := filepath.Join(cachePath, cacheKey)
		if data, contentType, err := readFromCache(cacheFile); err == nil {
			// For videos, set Accept-Ranges header
			if isVideo {
				c.Set("Accept-Ranges", "bytes")
			}
			c.Set("X-Cache", "HIT")
			c.Set("Cache-Control", "public, max-age=31536000")
			c.Set("Content-Type", contentType)
			return c.Send(data)
		}
	}

	// Fetch from S3
	ctx := context.Background()
	data, contentType, err := Download(ctx, path)
	if err != nil {
		log.Error("Failed to download from S3: %v", err)
		return c.Status(404).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	// Check if transformation is needed
	needsTransform := format != "" || sizeStr != ""
	isImage := isImageContentType(contentType)

	if needsTransform && isImage {
		// Decode image
		img, err := decodeImage(data, contentType)
		if err != nil {
			log.Error("Failed to decode image: %v", err)
			// Return original on error
			c.Set("Content-Type", contentType)
			c.Set("Cache-Control", "public, max-age=31536000")
			return c.Send(data)
		}

		// Resize if needed
		if sizeStr != "" {
			img = resizeImage(img, sizeStr)
		}

		// Convert format
		outputFormat := format
		if outputFormat == "" {
			outputFormat = getFormatFromContentType(contentType)
		}

		// Encode
		outputData, outputContentType, err := encodeImage(img, outputFormat, qualityInt)
		if err != nil {
			log.Error("Failed to encode image: %v", err)
			c.Set("Content-Type", contentType)
			c.Set("Cache-Control", "public, max-age=31536000")
			return c.Send(data)
		}

		// Cache the result
		if cacheEnabled {
			cacheFile := filepath.Join(cachePath, cacheKey)
			go saveToCache(cacheFile, outputData, outputContentType)
		}

		c.Set("X-Cache", "MISS")
		c.Set("Cache-Control", "public, max-age=31536000")
		c.Set("Content-Type", outputContentType)
		return c.Send(outputData)
	}

	// No transformation needed, return original
	if cacheEnabled {
		cacheFile := filepath.Join(cachePath, cacheKey)
		go saveToCache(cacheFile, data, contentType)
	}

	// For videos, set Accept-Ranges header
	if isVideo {
		c.Set("Accept-Ranges", "bytes")
	}

	c.Set("X-Cache", "MISS")
	c.Set("Cache-Control", "public, max-age=31536000")
	c.Set("Content-Type", contentType)
	return c.Send(data)
}

// isVideoPath checks if the path is a video file
func isVideoPath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".mp4" || ext == ".webm" || ext == ".mov" || ext == ".avi" || ext == ".mkv"
}

// handleVideoRangeRequest handles HTTP Range requests for video seeking
func handleVideoRangeRequest(c *fiber.Ctx, path string, rangeHeader string) error {
	ctx := context.Background()

	// First, get the file info to know total size
	info, err := GetObjectInfo(ctx, path)
	if err != nil {
		log.Error("Failed to get object info: %v", err)
		return c.Status(404).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	totalSize := info.Size
	contentType := info.ContentType
	if contentType == "" {
		contentType = getContentTypeFromExt(filepath.Ext(path))
	}

	// Parse Range header: "bytes=0-" or "bytes=0-1023"
	rangeStart, rangeEnd := parseRangeHeader(rangeHeader, totalSize)

	// Create the S3 range header format
	s3Range := fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd)

	// Download the range from S3
	body, _, _, _, err := DownloadRange(ctx, path, s3Range)
	if err != nil {
		log.Error("Failed to download range from S3: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to fetch video range",
		})
	}
	defer body.Close()

	// Read the entire chunk into memory first to avoid streaming issues
	// This prevents nginx timeout when streaming from S3 through Fiber
	data, err := io.ReadAll(body)
	if err != nil {
		log.Error("Failed to read video data: %v", err)
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to read video data",
		})
	}

	// Set response headers for partial content
	c.Set("Accept-Ranges", "bytes")
	c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rangeStart, rangeEnd, totalSize))
	c.Set("Content-Length", strconv.FormatInt(int64(len(data)), 10))
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=31536000")

	// Return 206 Partial Content with the data
	return c.Status(206).Send(data)
}

// parseRangeHeader parses HTTP Range header and returns start and end bytes
// Only limits OPEN-ENDED ranges (bytes=X-) to prevent timeout
// Explicit ranges (bytes=X-Y) are honored exactly for proper video seeking
func parseRangeHeader(rangeHeader string, totalSize int64) (int64, int64) {
	// Maximum chunk size for open-ended ranges only
	const maxChunkSize int64 = 5 * 1024 * 1024

	// Format: "bytes=0-" or "bytes=0-1023" or "bytes=-500" (last 500 bytes)
	rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")

	parts := strings.Split(rangeHeader, "-")
	if len(parts) != 2 {
		// Invalid format, return first chunk
		end := maxChunkSize - 1
		if end >= totalSize {
			end = totalSize - 1
		}
		return 0, end
	}

	var start, end int64

	// Handle suffix range: "-500" means last 500 bytes
	if parts[0] == "" {
		if suffix, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
			start = totalSize - suffix
			if start < 0 {
				start = 0
			}
			// Suffix ranges are typically small, return full requested range
			return start, totalSize - 1
		}
		// Parse error, return first chunk
		end = maxChunkSize - 1
		if end >= totalSize {
			end = totalSize - 1
		}
		return 0, end
	}

	// Parse start
	var err error
	start, err = strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		start = 0
	}
	if start < 0 {
		start = 0
	}
	if start >= totalSize {
		start = totalSize - 1
	}

	// Parse end
	if parts[1] == "" {
		// OPEN-ENDED range: "0-" or "1000-" means from start to end
		// THIS is where we limit to prevent timeout on large files
		end = start + maxChunkSize - 1
		if end >= totalSize {
			end = totalSize - 1
		}
	} else {
		// EXPLICIT range: "0-1023" means bytes 0 to 1023
		// Honor exactly what was requested - DON'T limit
		// This is critical for video seeking to work correctly
		end, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			end = start + maxChunkSize - 1
		}
		// Cap at file size but don't reduce below requested
		if end >= totalSize {
			end = totalSize - 1
		}
	}

	// Final validation
	if start > end {
		start = 0
		end = maxChunkSize - 1
		if end >= totalSize {
			end = totalSize - 1
		}
	}

	return start, end
}

// generateCacheKey generates a unique cache key for the request
func generateCacheKey(path, format, size string, quality int) string {
	key := fmt.Sprintf("%s_%s_%s_%d", path, format, size, quality)
	hash := md5.Sum([]byte(key))
	ext := ".bin"
	if format != "" {
		ext = "." + format
	}
	return fmt.Sprintf("%x%s", hash, ext)
}

// readFromCache reads data from cache
func readFromCache(cacheFile string) ([]byte, string, error) {
	// Check if cache file exists and is not too old
	info, err := os.Stat(cacheFile)
	if err != nil {
		return nil, "", err
	}

	// Check if cache is older than configured duration
	if time.Since(info.ModTime()) > cacheDuration {
		os.Remove(cacheFile)
		return nil, "", fmt.Errorf("cache expired")
	}

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, "", err
	}

	// Determine content type from file extension
	ext := filepath.Ext(cacheFile)
	contentType := getContentTypeFromExt(ext)

	return data, contentType, nil
}

// saveToCache saves data to cache
func saveToCache(cacheFile string, data []byte, contentType string) {
	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		log.Error("Failed to create cache directory: %v", err)
		return
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		log.Error("Failed to write cache file: %v", err)
	}
}

// isImageContentType checks if content type is an image
func isImageContentType(contentType string) bool {
	return strings.HasPrefix(contentType, "image/")
}

// getFormatFromContentType extracts format from content type
func getFormatFromContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "jpg"
	}
}

// getContentTypeFromExt returns content type from file extension
func getContentTypeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".pdf":
		return "application/pdf"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

// decodeImage decodes image data
func decodeImage(data []byte, contentType string) (image.Image, error) {
	reader := bytes.NewReader(data)

	switch contentType {
	case "image/webp":
		return webp.Decode(reader)
	default:
		img, _, err := image.Decode(reader)
		return img, err
	}
}

// resizeImage resizes an image based on size string
// Formats: 256x- (width only), -x256 (height only), 256x256 (both)
func resizeImage(img image.Image, sizeStr string) image.Image {
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	parts := strings.Split(sizeStr, "x")
	if len(parts) != 2 {
		return img
	}

	var targetWidth, targetHeight int

	if parts[0] != "" && parts[0] != "-" {
		w, err := strconv.Atoi(parts[0])
		if err == nil && w > 0 {
			targetWidth = w
		}
	}

	if parts[1] != "" && parts[1] != "-" {
		h, err := strconv.Atoi(parts[1])
		if err == nil && h > 0 {
			targetHeight = h
		}
	}

	// Calculate dimensions maintaining aspect ratio
	if targetWidth > 0 && targetHeight == 0 {
		// Width only - calculate height
		targetHeight = origHeight * targetWidth / origWidth
	} else if targetHeight > 0 && targetWidth == 0 {
		// Height only - calculate width
		targetWidth = origWidth * targetHeight / origHeight
	} else if targetWidth == 0 && targetHeight == 0 {
		// No valid dimensions
		return img
	}

	// Don't upscale
	if targetWidth > origWidth {
		targetWidth = origWidth
		targetHeight = origHeight * targetWidth / origWidth
	}
	if targetHeight > origHeight {
		targetHeight = origHeight
		targetWidth = origWidth * targetHeight / origHeight
	}

	// Create resized image
	resized := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

	return resized
}

// encodeImage encodes an image to the specified format
func encodeImage(img image.Image, format string, quality int) ([]byte, string, error) {
	var buf bytes.Buffer
	var contentType string

	switch strings.ToLower(format) {
	case "jpg", "jpeg":
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return nil, "", err
		}
		contentType = "image/jpeg"

	case "png":
		err := png.Encode(&buf, img)
		if err != nil {
			return nil, "", err
		}
		contentType = "image/png"

	case "webp":
		// Go doesn't have native webp encoder, fallback to jpeg
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return nil, "", err
		}
		contentType = "image/jpeg"

	default:
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
		if err != nil {
			return nil, "", err
		}
		contentType = "image/jpeg"
	}

	return buf.Bytes(), contentType, nil
}

// ClearCache clears the media cache
func ClearCache() error {
	if cachePath == "" {
		return nil
	}

	entries, err := os.ReadDir(cachePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		os.RemoveAll(filepath.Join(cachePath, entry.Name()))
	}

	return nil
}

// GetCacheStats returns cache statistics
type CacheStats struct {
	TotalFiles int64   `json:"total_files"`
	TotalSize  int64   `json:"total_size"`
	OldestFile string  `json:"oldest_file"`
	NewestFile string  `json:"newest_file"`
}

func GetCacheStats() (*CacheStats, error) {
	if cachePath == "" {
		return nil, fmt.Errorf("cache not initialized")
	}

	stats := &CacheStats{}
	var oldestTime, newestTime time.Time

	err := filepath.Walk(cachePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			stats.TotalFiles++
			stats.TotalSize += info.Size()

			modTime := info.ModTime()
			if oldestTime.IsZero() || modTime.Before(oldestTime) {
				oldestTime = modTime
				stats.OldestFile = info.Name()
			}
			if newestTime.IsZero() || modTime.After(newestTime) {
				newestTime = modTime
				stats.NewestFile = info.Name()
			}
		}
		return nil
	})

	return stats, err
}

// RegisterMediaProxy registers the media proxy routes
func RegisterMediaProxy(router fiber.Router) {
	// Initialize media proxy
	if err := InitMediaProxy(); err != nil {
		log.Warning("Failed to initialize media proxy: %v", err)
	}

	// Register routes
	router.Get("/media/*", MediaProxyHandler)
	router.Get("/api/admin/storage/cache/stats", GetCacheStatsHandler)
	router.Delete("/api/admin/storage/cache", ClearCacheHandler)
}

// GetCacheStatsHandler returns cache statistics
func GetCacheStatsHandler(c *fiber.Ctx) error {
	stats, err := GetCacheStats()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(stats)
}

// ClearCacheHandler clears the cache
func ClearCacheHandler(c *fiber.Ctx) error {
	if err := ClearCache(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Cache cleared",
	})
}

// Helper function to register decoders
func init() {
	// PNG decoder is registered by default
	// JPEG decoder is registered by default
	image.RegisterFormat("png", "\x89PNG", png.Decode, png.DecodeConfig)
}
