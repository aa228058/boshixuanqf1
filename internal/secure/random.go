package secure

import (
 "crypto/rand"
 "math/big"
)

// IntN returns a cryptographically secure integer in [0,n).
func IntN(n int) int {
 if n <= 0 { return 0 }
 v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
 if err != nil { return 0 }
 return int(v.Int64())
}

// Int63n returns a cryptographically secure integer in [0,n).
func Int63n(n int64) int64 {
 if n <= 0 { return 0 }
 v, err := rand.Int(rand.Reader, big.NewInt(n))
 if err != nil { return 0 }
 return v.Int64()
}

// Int64 returns a positive cryptographically secure identifier.
func Int64() int64 { return Int63n(1<<63-1) + 1 }
