package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

// Sealer encrypts small secrets, like invitation tokens, with AES-256-GCM so
// they can be stored and read back later. GCM also detects tampering.
type Sealer struct {
	aead cipher.AEAD
}

// NewSealer takes a 32-byte key written as 64 hex characters. Generate one
// with: openssl rand -hex 32
func NewSealer(hexKey string) (*Sealer, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("key must be 64 hex characters (32 bytes)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Sealer{aead: aead}, nil
}

// Seal encrypts plaintext. Each call uses a fresh random nonce, which is
// stored in front of the ciphertext.
func (s *Sealer) Seal(plaintext string) (string, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := s.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

// Open decrypts a value from Seal. It fails if the value was changed or was
// sealed with a different key.
func (s *Sealer) Open(sealed string) (string, error) {
	data, err := base64.RawStdEncoding.DecodeString(sealed)
	if err != nil {
		return "", err
	}
	n := s.aead.NonceSize()
	if len(data) < n {
		return "", errors.New("sealed value is too short")
	}
	plain, err := s.aead.Open(nil, data[:n], data[n:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
