package continuity

import (
	"github.com/opencontainers/go-digest"
	"io"
)

// Digester produces a digest for a given read stream
type Digester interface {
	Digest(io.Reader) (digest.Digest, error)
}

// ContentProvider produces a read stream for a given digest
type ContentProvider interface {
	Reader(digest.Digest) (io.ReadCloser, error)
}

// uniqifyDigests sorts and uniqifies the provided digest, ensuring that the
// digests are not repeated and no two digests with the same algorithm have
// different values. Because a stable sort is used, this has the effect of
// "zipping" digest collections from multiple resources.

// digestsMatch compares the two sets of digests to see if they match.
