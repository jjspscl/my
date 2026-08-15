package infrastructure

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
)

// SecretBox encrypts and decrypts provider/connector credentials with
// AES-256-GCM. The key is derived from the environment-held master key
// (MY_LLM_MASTER_KEY); ciphertext carries a key version for rotation.
type SecretBox struct {
	key     []byte
	version int
}

const CurrentKeyVersion = 1

// NewSecretBox derives the encryption key from the master key string. An
// empty master key makes every operation fail closed.
func NewSecretBox(masterKey string) (*SecretBox, error) {
	if masterKey == "" {
		return nil, fmt.Errorf("master key is not configured")
	}
	sum := sha256.Sum256([]byte(masterKey))
	return &SecretBox{key: sum[:], version: CurrentKeyVersion}, nil
}

// Encrypt returns "v{version}.{nonce}.{ciphertext}" base64-encoded.
func (s *SecretBox) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return fmt.Sprintf("v%d.%s", s.version, base64.RawStdEncoding.EncodeToString(sealed)), nil
}

// Decrypt reverses Encrypt. Only the current key version is supported; older
// versions fail loudly so operators rotate keys explicitly.
func (s *SecretBox) Decrypt(blob string) (string, error) {
	var version int
	var payload string
	if _, err := fmt.Sscanf(blob, "v%d.%s", &version, &payload); err != nil {
		return "", fmt.Errorf("malformed ciphertext: %w", err)
	}
	if version != s.version {
		return "", fmt.Errorf("ciphertext key version %d does not match current version %d", version, s.version)
	}
	raw, err := base64.RawStdEncoding.DecodeString(payload)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("new gcm: %w", err)
	}
	if len(raw) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return string(plaintext), nil
}
