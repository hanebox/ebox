package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

const payloadPrefix = "v1:"

var (
	ErrNotInitialized = errors.New("crypto: not initialized")

	mu   sync.RWMutex
	aead cipher.AEAD
)

func Init(appKey string) error {
	if strings.TrimSpace(appKey) == "" {
		return errors.New("crypto: APP_KEY must not be empty")
	}

	key := sha256.Sum256([]byte(appKey))

	block, err := aes.NewCipher(key[:])
	if err != nil {
		return err
	}

	instance, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	mu.Lock()
	aead = instance
	mu.Unlock()

	return nil
}

func getAEAD() (cipher.AEAD, error) {
	mu.RLock()
	defer mu.RUnlock()

	if aead == nil {
		return nil, ErrNotInitialized
	}

	return aead, nil
}

func Encrypt(plaintext string) (string, error) {
	instance, err := getAEAD()
	if err != nil {
		return "", err
	}

	nonce := make([]byte, instance.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	sealed := instance.Seal(nil, nonce, []byte(plaintext), nil)

	return payloadPrefix + base64.RawURLEncoding.EncodeToString(append(nonce, sealed...)), nil
}

func Decrypt(encoded string) (string, error) {
	instance, err := getAEAD()
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(encoded, payloadPrefix) {
		return "", errors.New("crypto: unknown payload format")
	}

	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(encoded, payloadPrefix))
	if err != nil {
		return "", err
	}

	nonceSize := instance.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("crypto: payload is too short")
	}

	plaintext, err := instance.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))

	return hex.EncodeToString(sum[:])
}

func RandomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func Mask(value string) string {
	if value == "" {
		return ""
	}

	return fmt.Sprintf("%s***%s", value[:1], value[len(value)-1:])
}
