package platforms

import (
	"runtime"

	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// DefaultSpec returns the current platform's default platform specification.
func DefaultSpec() specs.Platform {
	return specs.Platform{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		// The Variant field will be empty if arch != ARM.
		Variant: cpuVariant(),
	}
}

// Default returns the default matcher for the platform.
func Default() MatchComparer {
	return Only(DefaultSpec())
}
