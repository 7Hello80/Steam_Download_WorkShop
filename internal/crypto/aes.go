package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

var derivedKey []byte

// InitKey derives the AES-256 key from the provided secret.
// Must be called once at startup before Encrypt/Decrypt are used.
func InitKey(secret string) error {
	if secret == "" {
		return fmt.Errorf("AES encryption key must not be empty (set aes_key in config.yaml or AES_KEY env var)")
	}
	h := sha256.Sum256([]byte(secret))
	derivedKey = h[:]
	return nil
}

// Encrypt encrypts plaintext using AES-256-CBC and returns hex-encoded ciphertext.
func Encrypt(plaintext string) (string, error) {
	if derivedKey == nil {
		return "", errors.New("crypto not initialized: call InitKey first")
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", err
	}

	// PKCS7 padding
	padLen := aes.BlockSize - len(plaintext)%aes.BlockSize
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}

	// Random IV
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", err
	}

	ciphertext := make([]byte, len(padded))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, padded)

	// Prepend IV to ciphertext and hex-encode
	result := make([]byte, 0, len(iv)+len(ciphertext))
	result = append(result, iv...)
	result = append(result, ciphertext...)

	return hex.EncodeToString(result), nil
}

// Decrypt decrypts a hex-encoded ciphertext using AES-256-CBC.
func Decrypt(hexStr string) (string, error) {
	if derivedKey == nil {
		return "", errors.New("crypto not initialized: call InitKey first")
	}

	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}

	if len(data) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", err
	}

	iv := data[:aes.BlockSize]
	ciphertext := data[aes.BlockSize:]

	if len(ciphertext)%aes.BlockSize != 0 {
		return "", errors.New("ciphertext is not a multiple of block size")
	}

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove PKCS7 padding
	if len(plaintext) == 0 {
		return "", errors.New("plaintext is empty")
	}
	padLen := int(plaintext[len(plaintext)-1])
	if padLen > aes.BlockSize || padLen == 0 {
		return "", errors.New("invalid padding")
	}
	for i := len(plaintext) - padLen; i < len(plaintext); i++ {
		if plaintext[i] != byte(padLen) {
			return "", errors.New("invalid padding")
		}
	}

	return string(plaintext[:len(plaintext)-padLen]), nil
}
