package leases

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// WithRandomID sets the lease ID to a random unique value
func WithRandomID() Opt {
	return func(l *Lease) error {
		t := time.Now()
		var b [3]byte
		rand.Read(b[:])
		l.ID = fmt.Sprintf("%d-%s", t.Nanosecond(), base64.URLEncoding.EncodeToString(b[:]))
		return nil
	}
}

// WithID sets the ID for the lease
func WithID(id string) Opt {
	return func(l *Lease) error {
		l.ID = id
		return nil
	}
}
