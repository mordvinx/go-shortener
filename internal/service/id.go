package service

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

// GenerateID combines a random prefix with a unique sequence number.
// The caller must provide a different number for each new link.
func GenerateID(number uint64) (string, error) {
	var random [3]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("generate random prefix: %w", err)
	}

	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], number)
	start := 0
	for start < len(counter)-1 && counter[start] == 0 {
		start++
	}

	encoding := base64.RawURLEncoding
	return encoding.EncodeToString(random[:]) + encoding.EncodeToString(counter[start:]), nil
}
