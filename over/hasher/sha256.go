package hasher

import (
	"hash"

	sha256simd "github.com/minio/sha256-simd"
)

func NewSHA256() hash.Hash {
	// An equivalent of sha256-simd is expected to be merged in Go 1.20 (1.21?): https://go-review.googlesource.com/c/go/+/408795/1
	return sha256simd.New()
}
