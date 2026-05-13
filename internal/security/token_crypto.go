package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

var (
	// ErrEncryptionKeyMissing is returned when the encryption key env var is not set.
	ErrEncryptionKeyMissing = errors.New("MARKETPLACE_TOKEN_ENCRYPTION_KEY environment variable is not set")
	// ErrEncryptionKeyInvalid is returned when the key is not the correct length.
	ErrEncryptionKeyInvalid = errors.New("MARKETPLACE_TOKEN_ENCRYPTION_KEY must be exactly 32 bytes (use a 32-character string or 44-char base64 of 32 bytes)")
	// ErrDecryptionFailed is returned when decryption cannot be performed.
	ErrDecryptionFailed = errors.New("failed to decrypt token: ciphertext is invalid or corrupted")
)

// getEncryptionKey reads and validates the AES-256 encryption key from the environment.
// The key must be exactly 32 bytes after decoding.
// Supports both raw 32-byte strings and base64-encoded 32-byte values.
func getEncryptionKey() ([]byte, error) {
	keyStr := os.Getenv("MARKETPLACE_TOKEN_ENCRYPTION_KEY")
	if keyStr == "" {
		return nil, ErrEncryptionKeyMissing
	}

	// Try raw bytes first (exactly 32 chars)
	if len(keyStr) == 32 {
		return []byte(keyStr), nil
	}

	// Try base64 decoding
	decoded, err := base64.StdEncoding.DecodeString(keyStr)
	if err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	// Try base64 URL-safe decoding
	decoded, err = base64.URLEncoding.DecodeString(keyStr)
	if err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	return nil, ErrEncryptionKeyInvalid
}

// EncryptToken encrypts a plaintext token using AES-256-GCM with a random nonce.
// The result is base64-encoded for safe database storage.
// SECURITY: Never log the plaintext input or the encryption key.
func EncryptToken(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}

	key, err := getEncryptionKey()
	if err != nil {
		return "", fmt.Errorf("token encryption failed: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("token encryption failed: could not create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("token encryption failed: could not create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("token encryption failed: could not generate nonce: %w", err)
	}

	// Seal appends the encrypted data to the nonce, so the nonce is stored alongside
	cipherText := aesGCM.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// DecryptToken decrypts a base64-encoded AES-256-GCM encrypted token.
// SECURITY: Never log the decrypted output.
func DecryptToken(cipherText string) (string, error) {
	if cipherText == "" {
		return "", nil
	}

	key, err := getEncryptionKey()
	if err != nil {
		return "", fmt.Errorf("token decryption failed: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", fmt.Errorf("token decryption failed: invalid base64: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("token decryption failed: could not create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("token decryption failed: could not create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", ErrDecryptionFailed
	}

	nonce, encrypted := data[:nonceSize], data[nonceSize:]
	plainText, err := aesGCM.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plainText), nil
}

// ValidateEncryptionKey checks if the encryption key is configured and valid.
// Returns nil if valid, or a descriptive error.
// This does NOT require the key to be set — it only validates if present.
func ValidateEncryptionKey() error {
	keyStr := os.Getenv("MARKETPLACE_TOKEN_ENCRYPTION_KEY")
	if keyStr == "" {
		return nil // Key is optional until token operations are needed
	}

	_, err := getEncryptionKey()
	return err
}
