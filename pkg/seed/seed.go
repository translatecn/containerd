// Package seed provides an initializer for the global [math/rand] seed.
//
// Deprecated: Do not rely on the global seed.
package seed

import (
	"math/rand"
	"time"
)

// WithTimeAndRand seeds the global math rand generator with nanoseconds
// XOR'ed with a crypto component if available for uniqueness.
//
// Deprecated: Do not rely on the global seed.
func WithTimeAndRand() {
	var (
		b [4]byte
		u int64
	)

	tryReadRandom(b[:])

	// Set higher 32 bits, bottom 32 will be set with nanos
	u |= (int64(b[0]) << 56) | (int64(b[1]) << 48) | (int64(b[2]) << 40) | (int64(b[3]) << 32)

	rand.Seed(u ^ time.Now().UnixNano())
}
