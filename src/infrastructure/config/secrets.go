package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

var (
	// ErrSecretNotFound indicates a secret was not found
	ErrSecretNotFound = errors.New("secret not found")
	// ErrEmptySecretKey indicates an empty secret key
	ErrEmptySecretKey = errors.New("empty secret key")
	// ErrEmptySecretValue indicates an empty secret value
	ErrEmptySecretValue = errors.New("empty secret value")
	// ErrWeakSecret indicates a secret doesn't meet strength requirements
	ErrWeakSecret = errors.New("weak secret")
	// ErrSameSecret indicates new secret is same as old secret
	ErrSameSecret = errors.New("new secret same as old secret")
	// ErrEncryptionFailed indicates encryption failed
	ErrEncryptionFailed = errors.New("encryption failed")
	// ErrDecryptionFailed indicates decryption failed
	ErrDecryptionFailed = errors.New("decryption failed")
)

// SecretStore manages application secrets
type SecretStore struct {
	secrets map[string]string
	mu      sync.RWMutex
	key     []byte
}

var globalSecretStore *SecretStore
var secretStoreMutex sync.Mutex

// GetSecretStore returns the global secret store instance
func GetSecretStore() *SecretStore {
	secretStoreMutex.Lock()
	defer secretStoreMutex.Unlock()

	if globalSecretStore == nil {
		globalSecretStore = NewSecretStore()
	}

	return globalSecretStore
}

// NewSecretStore creates a new secret store
func NewSecretStore() *SecretStore {
	// Use a default encryption key (in production, should come from secure source)
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		// Fallback to a deterministic key for testing
		copy(key, []byte("default-encryption-key-32-chars!!"))
	}

	return &SecretStore{
		secrets: make(map[string]string),
		key:     key,
	}
}

// LoadSecrets loads secrets from environment variables
func LoadSecrets() error {
	store := GetSecretStore()

	requiredSecrets := []string{
		"LINKGEN_MONGODB_PASSWORD",
		"LINKGEN_LLM_API_KEY",
		"LINKGEN_LINKEDIN_CLIENT_SECRET",
	}

	for _, secretKey := range requiredSecrets {
		value := os.Getenv(secretKey)
		if value == "" {
			return fmt.Errorf("%w: %s", ErrSecretNotFound, secretKey)
		}

		// Store without prefix for easier access
		key := strings.TrimPrefix(secretKey, "LINKGEN_")
		key = strings.ToLower(key)

		if err := store.SetSecret(key, value); err != nil {
			return err
		}
	}

	return nil
}

// GetSecret retrieves a secret by key
func (s *SecretStore) GetSecret(key string) (string, error) {
	if key == "" {
		return "", ErrEmptySecretKey
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	value, exists := s.secrets[key]
	if !exists {
		return "", fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	}

	return value, nil
}

// SetSecret stores or updates a secret
func (s *SecretStore) SetSecret(key, value string) error {
	if key == "" {
		return ErrEmptySecretKey
	}
	if value == "" {
		return ErrEmptySecretValue
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.secrets[key] = value
	return nil
}

// RotateSecret rotates a secret to a new value
func (s *SecretStore) RotateSecret(key, oldSecret, newSecret string) error {
	if oldSecret == newSecret {
		return ErrSameSecret
	}
	if newSecret == "" {
		return ErrEmptySecretValue
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	currentSecret, exists := s.secrets[key]
	if !exists || currentSecret != oldSecret {
		return fmt.Errorf("old secret doesn't match current secret")
	}

	s.secrets[key] = newSecret
	return nil
}

// MaskSecrets masks sensitive information in a string
func MaskSecrets(input string, secretKeys []string) string {
	result := input

	store := GetSecretStore()
	store.mu.RLock()
	defer store.mu.RUnlock()

	// Mask each secret
	for _, key := range secretKeys {
		secret, exists := store.secrets[key]
		if exists && secret != "" {
			result = strings.ReplaceAll(result, secret, "***")
		}
	}

	// Also mask MongoDB passwords in URIs
	result = maskMongoDBPassword(result)

	return result
}

// maskMongoDBPassword masks passwords in MongoDB URIs
func maskMongoDBPassword(uri string) string {
	if !strings.Contains(uri, "mongodb://") {
		return uri
	}

	return maskSecret(uri)
}

// EncryptSecret encrypts a secret using AES-GCM
func (s *SecretStore) EncryptSecret(plaintext string) (string, error) {
	if plaintext == "" {
		return "", ErrEmptySecretValue
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSecret decrypts an encrypted secret
func (s *SecretStore) DecryptSecret(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", ErrEmptySecretValue
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext too short", ErrDecryptionFailed)
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return string(plaintext), nil
}

// ValidateSecretStrength validates if a secret meets strength requirements
func ValidateSecretStrength(secret string, minLength int) error {
	if secret == "" {
		return ErrEmptySecretValue
	}
	if len(secret) < minLength {
		return fmt.Errorf("%w: secret must be at least %d characters, got %d", ErrWeakSecret, minLength, len(secret))
	}
	return nil
}

// LoadFromExternalSecretStore loads secrets from external stores (placeholder)
func LoadFromExternalSecretStore(storeType, secretPath string) error {
	if secretPath == "" {
		return errors.New("empty secret path")
	}

	switch storeType {
	case "vault", "aws", "azure":
		// Placeholder - would integrate with actual secret store
		return nil
	default:
		return fmt.Errorf("unsupported secret store type: %s", storeType)
	}
}
