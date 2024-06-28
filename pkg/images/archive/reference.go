package archive

import (
	"demo/pkg/reference"
	docker2 "demo/pkg/reference/docker"
	"fmt"
	"strings"

	"github.com/opencontainers/go-digest"
)

// FilterRefPrefix restricts references to having the given image
// prefix. Tag-only references will have the prefix prepended.
func FilterRefPrefix(image string) func(string) string {
	return refTranslator(image, true)
}

// AddRefPrefix prepends the given image prefix to tag-only references,
// while leaving returning full references unmodified.
func AddRefPrefix(image string) func(string) string {
	return refTranslator(image, false)
}

// refTranslator creates a reference which only has a tag or verifies
// a full reference.
func refTranslator(image string, checkPrefix bool) func(string) string {
	return func(ref string) string {
		if image == "" {
			return ""
		}
		// Check if ref is full reference
		if strings.ContainsAny(ref, "/:@") {
			// If not prefixed, don't include image
			if checkPrefix && !isImagePrefix(ref, image) {
				return ""
			}
			return ref
		}
		return image + ":" + ref
	}
}

func isImagePrefix(s, prefix string) bool {
	if !strings.HasPrefix(s, prefix) {
		return false
	}
	if len(s) > len(prefix) {
		switch s[len(prefix)] {
		case '/', ':', '@':
			// Prevent matching partial namespaces
		default:
			return false
		}
	}
	return true
}

func normalizeReference(ref string) (string, error) {
	// TODO: Replace this function to not depend on reference package
	normalized, err := docker2.ParseDockerRef(ref)
	if err != nil {
		return "", fmt.Errorf("normalize image ref %q: %w", ref, err)
	}

	return normalized.String(), nil
}

func familiarizeReference(ref string) (string, error) {
	named, err := docker2.ParseNormalizedNamed(ref)
	if err != nil {
		return "", fmt.Errorf("failed to parse %q: %w", ref, err)
	}
	named = docker2.TagNameOnly(named)

	return docker2.FamiliarString(named), nil
}

func ociReferenceName(name string) string {
	// OCI defines the reference name as only a tag excluding the
	// repository. The containerd annotation contains the full image name
	// since the tag is insufficient for correctly naming and referring to an
	// image
	var ociRef string
	if spec, err := reference.Parse(name); err == nil {
		ociRef = spec.Object
	} else {
		ociRef = name
	}

	return ociRef
}

// DigestTranslator creates a digest reference by adding the
// digest to an image name
func DigestTranslator(prefix string) func(digest.Digest) string {
	return func(dgst digest.Digest) string {
		return prefix + "@" + dgst.String()
	}
}
