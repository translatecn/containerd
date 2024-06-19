package continuity

import (
	driverpkg "demo/others/continuity/driver"
	"demo/others/continuity/pathdriver"
	"os"
	"path/filepath"
)

// Context represents a file system context for accessing resources. The
// responsibility of the context is to convert system specific resources to
// generic Resource objects. Most of this is safe path manipulation, as well
// as extraction of resource details.
type Context interface {
	Apply(Resource) error
	Verify(Resource) error
	Resource(string, os.FileInfo) (Resource, error)
	Walk(filepath.WalkFunc) error
}

// SymlinkPath is intended to give the symlink target value
// in a root context. Target and linkname are absolute paths
// not under the given root.
type SymlinkPath func(root, linkname, target string) (string, error)

// ContextOptions represents options to create a new context.
type ContextOptions struct {
	Digester   Digester
	Driver     driverpkg.Driver
	PathDriver pathdriver.PathDriver
	Provider   ContentProvider
}
