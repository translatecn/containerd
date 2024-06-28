package platforms

import (
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// NewMatcher returns the default Matcher for containerd
func newDefaultMatcher(platform specs.Platform) Matcher {
	return &matcher{
		Platform: Normalize(platform),
	}
}

func GetWindowsOsVersion() string {
	return ""
}
