package util

import (
	"crypto/rand"
	"encoding/hex"
)

func GenerateRandomID(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
