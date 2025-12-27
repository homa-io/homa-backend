package imageutil

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/getevo/evo/v2/lib/settings"
	"github.com/google/uuid"
	"golang.org/x/image/draw"
)

// GetStoragePath returns the storage path from settings
func GetStoragePath() string {
	path := settings.Get("STORAGE.PATH").String()
	if path == "" {
		path = "uploads"
	}
	return path
}

// GetAvatarSize returns the avatar size from settings (default 64)
func GetAvatarSize() int {
	size := settings.Get("STORAGE.AVATAR_SIZE").Int()
	if size <= 0 {
		size = 64
	}
	return size
}

// ProcessAvatarFromBase64 takes a base64 encoded image, resizes it to the configured size,
// and saves it to disk. Returns the relative URL path.
func ProcessAvatarFromBase64(base64Data string, subdir string) (string, error) {
	// Parse base64 data - format: data:image/jpeg;base64,/9j/4AAQSkZJRg...
	parts := strings.Split(base64Data, ",")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid base64 format")
	}

	// Get base64 data (second part after comma)
	imageData := parts[1]

	// Decode base64
	imageBytes, err := base64.StdEncoding.DecodeString(imageData)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Decode image (we always save as JPEG regardless of input format)
	img, _, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	// Get target size
	targetSize := GetAvatarSize()

	// Resize image to square (crop center and resize)
	resizedImg := resizeAndCropToSquare(img, targetSize)

	// Encode to JPEG (best for avatars - small file size)
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, resizedImg, &jpeg.Options{Quality: 85})
	if err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	// Generate filename
	filename := uuid.New().String() + ".jpg"

	// Get storage path
	storagePath := GetStoragePath()
	avatarDir := filepath.Join(storagePath, "avatars", subdir)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	filePath := filepath.Join(avatarDir, filename)
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Return relative URL path
	return "/uploads/avatars/" + subdir + "/" + filename, nil
}

// resizeAndCropToSquare takes an image, crops it to a center square, and resizes to targetSize
func resizeAndCropToSquare(img image.Image, targetSize int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Determine the crop rectangle (center square)
	var cropRect image.Rectangle
	if width > height {
		// Wider than tall - crop sides
		offset := (width - height) / 2
		cropRect = image.Rect(offset, 0, offset+height, height)
	} else if height > width {
		// Taller than wide - crop top/bottom
		offset := (height - width) / 2
		cropRect = image.Rect(0, offset, width, offset+width)
	} else {
		// Already square
		cropRect = bounds
	}

	// Create cropped image
	croppedSize := cropRect.Dx()
	cropped := image.NewRGBA(image.Rect(0, 0, croppedSize, croppedSize))
	draw.Copy(cropped, image.Point{}, img, cropRect, draw.Src, nil)

	// Resize to target size
	resized := image.NewRGBA(image.Rect(0, 0, targetSize, targetSize))
	draw.CatmullRom.Scale(resized, resized.Bounds(), cropped, cropped.Bounds(), draw.Over, nil)

	return resized
}

// DeleteAvatar removes an avatar file from disk
func DeleteAvatar(avatarURL string) error {
	if avatarURL == "" {
		return nil
	}

	// Convert URL to file path
	// URL format: /uploads/avatars/clients/uuid.jpg
	// File path: {storage}/avatars/clients/uuid.jpg
	relativePath := strings.TrimPrefix(avatarURL, "/uploads/")
	if relativePath == avatarURL {
		// URL doesn't start with /uploads/, might be external URL
		return nil
	}

	storagePath := GetStoragePath()
	filePath := filepath.Join(storagePath, relativePath)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to delete
	}

	return os.Remove(filePath)
}

// Helper function to decode various image formats
func init() {
	// Register PNG decoder
	image.RegisterFormat("png", "\x89PNG", png.Decode, png.DecodeConfig)
	// JPEG is registered by default
}
