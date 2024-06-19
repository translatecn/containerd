// Package randutil provides utilities for [cyrpto/rand].
package randutil

import (
	"crypto/rand"
	"math/big"
)

// Int63n is similar to [math/rand.Int63n] but uses [crypto/rand.Reader] under the hood.
func Int63n(n int64) int64 {
	b, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		panic(err)
	}
	return b.Int64()
}

// Int63 is similar to [math/rand.Int63] but uses [crypto/rand.Reader] under the hood.

// Intn is similar to [math/rand.Intn] but uses [crypto/rand.Reader] under the hood.
func Intn(n int) int {
	return int(Int63n(int64(n)))
}

// Int is similar to [math/rand.Int] but uses [crypto/rand.Reader] under the hood.
