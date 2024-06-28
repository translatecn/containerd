package sys

import (
	"demo/pkg/my_mk"
	"os"
)

// MkdirAllWithACL is a wrapper for my_mk.MkdirAll on Unix systems.
func MkdirAllWithACL(path string, perm os.FileMode) error {
	return my_mk.MkdirAll(path, perm)
}
