package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateUUID はhyphenなし32文字のUUIDを生成する
func GenerateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}
	return hex.EncodeToString(b), nil
}
