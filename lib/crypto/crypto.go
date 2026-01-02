// Package crypto provides encryption utilities for securing sensitive data
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// EncryptAES256GCM encrypts plaintext using AES-256-GCM and returns base64-encoded ciphertext
// The encryption key is read from ENCRYPTION_KEY environment variable (must be 32 bytes)
func EncryptAES256GCM(plaintext string) (string, error) {
	key := []byte(os.Getenv("ENCRYPTION_KEY"))
	if len(key) == 0 {
		return "", fmt.Errorf("ENCRYPTION_KEY environment variable not set")
	}
	if len(key) != 32 {
		return "", fmt.Errorf("ENCRYPTION_KEY must be 32 bytes for AES-256, got %d bytes", len(key))
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM cipher mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt plaintext
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64-encoded ciphertext (includes nonce)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAES256GCM decrypts base64-encoded ciphertext using AES-256-GCM
// The encryption key is read from ENCRYPTION_KEY environment variable (must be 32 bytes)
func DecryptAES256GCM(ciphertext string) (string, error) {
	key := []byte(os.Getenv("ENCRYPTION_KEY"))
	if len(key) == 0 {
		return "", fmt.Errorf("ENCRYPTION_KEY environment variable not set")
	}
	if len(key) != 32 {
		return "", fmt.Errorf("ENCRYPTION_KEY must be 32 bytes for AES-256, got %d bytes", len(key))
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 ciphertext: %w", err)
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM cipher mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce from beginning of ciphertext
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short: expected at least %d bytes, got %d", nonceSize, len(data))
	}

	nonce, encryptedData := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a string appears to be encrypted (base64 with valid GCM format)
// This is used for backward compatibility when transitioning plaintext to encrypted storage
func IsEncrypted(data string) bool {
	if len(data) == 0 {
		return false
	}

	// Try to decode as base64
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return false
	}

	// Check if it has minimum length for GCM (nonce + at least one byte of data)
	if len(decoded) < 13 {
		return false
	}

	return true
}
