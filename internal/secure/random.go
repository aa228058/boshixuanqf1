package secure

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// IntN returns a cryptographically secure integer in [0,n).
func IntN(n int) int {
	if n <= 0 {
		return 0
	}
	return int(secureInt(int64(n)))
}

// Int63n returns a cryptographically secure integer in [0,n).
func Int63n(n int64) int64 {
	if n <= 0 {
		return 0
	}
	return secureInt(n)
}

// Int64 returns a positive cryptographically secure identifier.
func Int64() int64 {
	return secureInt(1<<63-1) + 1
}

// secureInt fails closed rather than silently falling back to predictable values
// if the operating system's cryptographic entropy source is unavailable.
func secureInt(n int64) int64 {
	v, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		panic(fmt.Errorf("crypto/rand unavailable: %w", err))
	}
	return v.Int64()
}
