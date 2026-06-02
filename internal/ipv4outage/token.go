package ipv4outage

import (
	"crypto/rand"
	"encoding/base64"
)

// NewToken returns an opaque Retry-Over-IPv6-Token value.
func NewToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
