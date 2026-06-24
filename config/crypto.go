package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"os"
	"os/exec"
	"strings"
)

func getSystemUUID() string {
	cmd := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "IOPlatformUUID") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				uuid := strings.TrimSpace(parts[1])
				uuid = strings.Trim(uuid, "\"")
				return uuid
			}
		}
	}
	return ""
}

func getEncryptionKey() []byte {
	uuid := getSystemUUID()
	if uuid == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			uuid = home
		} else {
			uuid = "kibuild_default_fallback_salt"
		}
	}
	hasher := sha256.New()
	hasher.Write([]byte(uuid))
	hasher.Write([]byte("kibuild_salt_value_2026"))
	return hasher.Sum(nil)
}

// EncryptKey encrypts a string using AES-GCM and returns a base64 encoded string.
func EncryptKey(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptKey decrypts a base64 encoded string using AES-GCM.
// Falls back to original string if decryption fails (so plaintext keys still work).
func DecryptKey(base64Ciphertext string) string {
	if base64Ciphertext == "" {
		return ""
	}
	ciphertext, err := base64.StdEncoding.DecodeString(base64Ciphertext)
	if err != nil {
		return base64Ciphertext
	}
	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return base64Ciphertext
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return base64Ciphertext
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return base64Ciphertext
	}
	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return base64Ciphertext
	}
	return string(plaintext)
}
