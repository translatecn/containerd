package util

import (
	"crypto/rand"
	"encoding/hex"
)

// GenerateID generates a random unique id.
func GenerateID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
