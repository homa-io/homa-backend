package storage

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CreateMultipartRequest represents a request to create a multipart upload
type CreateMultipartRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Prefix      string `json:"prefix"` // e.g., "kb", "avatars/users"
}

// CreateMultipartResponse represents the response for multipart upload creation
type CreateMultipartResponse struct {
	UploadID string `json:"uploadId"`
	Key      string `json:"key"`
}

// CreateMultipartUploadHandler handles the creation of a multipart upload
func CreateMultipartUploadHandler(c *fiber.Ctx) error {
	if !IsEnabled() {
		return c.Status(503).JSON(fiber.Map{
			"error": "S3 storage not enabled",
		})
	}

	var req CreateMultipartRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Filename == "" || req.ContentType == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Filename and contentType are required",
		})
	}

	// Generate a unique key
	ext := filepath.Ext(req.Filename)
	prefix := req.Prefix
	if prefix == "" {
		prefix = GetUploadPrefix() // Use configured default prefix
	}
	key := fmt.Sprintf("%s/%s%s", prefix, uuid.New().String(), ext)

	session, err := CreateMultipartUpload(c.Context(), key, req.ContentType)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to create multipart upload: %v", err),
		})
	}

	return c.JSON(CreateMultipartResponse{
		UploadID: session.UploadID,
		Key:      session.Key,
	})
}

// SignPartRequest represents a request to sign a part for upload
type SignPartRequest struct {
	Key        string `json:"key"`
	UploadID   string `json:"uploadId"`
	PartNumber int32  `json:"partNumber"`
}

// SignPartResponse represents the response with a presigned URL for part upload
type SignPartResponse struct {
	URL string `json:"url"`
}

// SignPartHandler handles signing a part for multipart upload
func SignPartHandler(c *fiber.Ctx) error {
	if !IsEnabled() {
		return c.Status(503).JSON(fiber.Map{
			"error": "S3 storage not enabled",
		})
	}

	var req SignPartRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Key == "" || req.UploadID == "" || req.PartNumber < 1 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Key, uploadId, and partNumber (>= 1) are required",
		})
	}

	url, err := GetPresignedUploadPartURL(c.Context(), req.Key, req.UploadID, req.PartNumber)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to sign part: %v", err),
		})
	}

	return c.JSON(SignPartResponse{URL: url})
}

// CompleteMultipartRequest represents a request to complete a multipart upload
type CompleteMultipartRequest struct {
	Key      string     `json:"key"`
	UploadID string     `json:"uploadId"`
	Parts    []PartInfo `json:"parts"`
}

// CompleteMultipartUploadHandler handles completing a multipart upload
func CompleteMultipartUploadHandler(c *fiber.Ctx) error {
	if !IsEnabled() {
		return c.Status(503).JSON(fiber.Map{
			"error": "S3 storage not enabled",
		})
	}

	var req CompleteMultipartRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Key == "" || req.UploadID == "" || len(req.Parts) == 0 {
		return c.Status(400).JSON(fiber.Map{
			"error": "Key, uploadId, and parts are required",
		})
	}

	if err := CompleteMultipartUpload(c.Context(), req.Key, req.UploadID, req.Parts); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to complete multipart upload: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"key":     req.Key,
		"success": true,
	})
}

// AbortMultipartRequest represents a request to abort a multipart upload
type AbortMultipartRequest struct {
	Key      string `json:"key"`
	UploadID string `json:"uploadId"`
}

// AbortMultipartUploadHandler handles aborting a multipart upload
func AbortMultipartUploadHandler(c *fiber.Ctx) error {
	if !IsEnabled() {
		return c.Status(503).JSON(fiber.Map{
			"error": "S3 storage not enabled",
		})
	}

	var req AbortMultipartRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Key == "" || req.UploadID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Key and uploadId are required",
		})
	}

	if err := AbortMultipartUpload(c.Context(), req.Key, req.UploadID); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to abort multipart upload: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// ListPartsHandler handles listing parts of a multipart upload
func ListPartsHandler(c *fiber.Ctx) error {
	if !IsEnabled() {
		return c.Status(503).JSON(fiber.Map{
			"error": "S3 storage not enabled",
		})
	}

	key := c.Query("key")
	uploadID := c.Query("uploadId")

	if key == "" || uploadID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Key and uploadId query parameters are required",
		})
	}

	parts, err := ListParts(c.Context(), key, uploadID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to list parts: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"parts": parts,
	})
}

// PresignUploadRequest represents a request for a presigned upload URL
type PresignUploadRequest struct {
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	Prefix      string `json:"prefix"` // e.g., "kb", "avatars/users"
}

// PresignUploadResponse represents the response with a presigned upload URL
type PresignUploadResponse struct {
	URL string `json:"url"`
	Key string `json:"key"`
}

// PresignUploadHandler handles generating a presigned URL for simple uploads (< 5MB)
func PresignUploadHandler(c *fiber.Ctx) error {
	if !IsEnabled() {
		return c.Status(503).JSON(fiber.Map{
			"error": "S3 storage not enabled",
		})
	}

	var req PresignUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Filename == "" || req.ContentType == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Filename and contentType are required",
		})
	}

	// Generate a unique key
	ext := filepath.Ext(req.Filename)
	if ext == "" {
		// Try to get extension from content type
		ext = getExtensionFromContentType(req.ContentType)
	}
	prefix := req.Prefix
	if prefix == "" {
		prefix = GetUploadPrefix() // Use configured default prefix
	}
	key := fmt.Sprintf("%s/%s%s", prefix, uuid.New().String(), ext)

	presignClient := NewPresignClient()
	if presignClient == nil {
		return c.Status(503).JSON(fiber.Map{
			"error": "S3 presign client not available",
		})
	}

	url, err := presignClient.GenerateUploadURL(c.Context(), key, req.ContentType, 1*time.Hour)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate presigned URL: %v", err),
		})
	}

	return c.JSON(PresignUploadResponse{
		URL: url,
		Key: key,
	})
}

// getExtensionFromContentType returns a file extension for a content type
func getExtensionFromContentType(contentType string) string {
	contentType = strings.ToLower(contentType)
	switch {
	case strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg"):
		return ".jpg"
	case strings.Contains(contentType, "png"):
		return ".png"
	case strings.Contains(contentType, "gif"):
		return ".gif"
	case strings.Contains(contentType, "webp"):
		return ".webp"
	case strings.Contains(contentType, "svg"):
		return ".svg"
	case strings.Contains(contentType, "pdf"):
		return ".pdf"
	case strings.Contains(contentType, "mp4"):
		return ".mp4"
	case strings.Contains(contentType, "webm"):
		return ".webm"
	case strings.Contains(contentType, "mp3"):
		return ".mp3"
	case strings.Contains(contentType, "wav"):
		return ".wav"
	default:
		return ""
	}
}
