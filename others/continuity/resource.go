package continuity

import (
	"github.com/opencontainers/go-digest"
	"os"
)

// TODO(stevvooe): A record based model, somewhat sketched out at the bottom
// of this file, will be more flexible. Another possibly is to tie the package
// interface directly to the protobuf type. This will have efficiency
// advantages at the cost coupling the nasty codegen types to the exported
// interface.

type Resource interface {
	// Path provides the primary resource path relative to the bundle root. In
	// cases where resources have more than one path, such as with hard links,
	// this will return the primary path, which is often just the first entry.
	Path() string

	// Mode returns the
	Mode() os.FileMode

	UID() int64
	GID() int64
}

// ByPath provides the canonical sort order for a set of resources. Use with
// sort.Stable for deterministic sorting.
type ByPath []Resource

func (bp ByPath) Len() int           { return len(bp) }
func (bp ByPath) Swap(i, j int)      { bp[i], bp[j] = bp[j], bp[i] }
func (bp ByPath) Less(i, j int) bool { return bp[i].Path() < bp[j].Path() }

type XAttrer interface {
	XAttrs() map[string][]byte
}

// Hardlinkable is an interface that a resource type satisfies if it can be a
// hardlink target.
type Hardlinkable interface {
	// Paths returns all paths of the resource, including the primary path
	// returned by Resource.Path. If len(Paths()) > 1, the resource is a hard
	// link.
	Paths() []string
}

type RegularFile interface {
	Resource
	XAttrer
	Hardlinkable

	Size() int64
	Digests() []digest.Digest
}

// Merge two or more Resources into new file. Typically, this should be
// used to merge regular files as hardlinks. If the files are not identical,
// other than Paths and Digests, the merge will fail and an error will be
// returned.

type Directory interface {
	Resource
	XAttrer

	// Directory is a no-op method to identify directory objects by interface.
	Directory()
}

type SymLink interface {
	Resource

	// Target returns the target of the symlink contained in the .
	Target() string
}

type NamedPipe interface {
	Resource
	Hardlinkable
	XAttrer

	// Pipe is a no-op method to allow consistent resolution of NamedPipe
	// interface.
	Pipe()
}

type Device interface {
	Resource
	Hardlinkable
	XAttrer

	Major() uint64
	Minor() uint64
}

type resource struct {
	paths    []string
	mode     os.FileMode
	uid, gid int64
	xattrs   map[string][]byte
}

var _ Resource = &resource{}

func (r *resource) Path() string {
	if len(r.paths) < 1 {
		return ""
	}

	return r.paths[0]
}

func (r *resource) Mode() os.FileMode {
	return r.mode
}

func (r *resource) UID() int64 {
	return r.uid
}

func (r *resource) GID() int64 {
	return r.gid
}

type regularFile struct {
	resource
	size    int64
	digests []digest.Digest
}

var _ RegularFile = &regularFile{}

// newRegularFile returns the RegularFile, using the populated base resource
// and one or more digests of the content.

func (rf *regularFile) Paths() []string {
	paths := make([]string, len(rf.paths))
	copy(paths, rf.paths)
	return paths
}

func (rf *regularFile) Size() int64 {
	return rf.size
}

func (rf *regularFile) Digests() []digest.Digest {
	digests := make([]digest.Digest, len(rf.digests))
	copy(digests, rf.digests)
	return digests
}

func (rf *regularFile) XAttrs() map[string][]byte {
	xattrs := make(map[string][]byte, len(rf.xattrs))

	for attr, value := range rf.xattrs {
		xattrs[attr] = append(xattrs[attr], value...)
	}

	return xattrs
}

type directory struct {
	resource
}

var _ Directory = &directory{}

func (d *directory) Directory() {}

func (d *directory) XAttrs() map[string][]byte {
	xattrs := make(map[string][]byte, len(d.xattrs))

	for attr, value := range d.xattrs {
		xattrs[attr] = append(xattrs[attr], value...)
	}

	return xattrs
}

type symLink struct {
	resource
	target string
}

var _ SymLink = &symLink{}

func (l *symLink) Target() string {
	return l.target
}

type namedPipe struct {
	resource
}

var _ NamedPipe = &namedPipe{}

func (np *namedPipe) Pipe() {}

func (np *namedPipe) Paths() []string {
	paths := make([]string, len(np.paths))
	copy(paths, np.paths)
	return paths
}

func (np *namedPipe) XAttrs() map[string][]byte {
	xattrs := make(map[string][]byte, len(np.xattrs))

	for attr, value := range np.xattrs {
		xattrs[attr] = append(xattrs[attr], value...)
	}

	return xattrs
}

type device struct {
	resource
	major, minor uint64
}

var _ Device = &device{}

func (d *device) Paths() []string {
	paths := make([]string, len(d.paths))
	copy(paths, d.paths)
	return paths
}

func (d *device) XAttrs() map[string][]byte {
	xattrs := make(map[string][]byte, len(d.xattrs))

	for attr, value := range d.xattrs {
		xattrs[attr] = append(xattrs[attr], value...)
	}

	return xattrs
}

func (d device) Major() uint64 {
	return d.major
}

func (d device) Minor() uint64 {
	return d.minor
}

// toProto converts a resource to a protobuf record. We'd like to push this
// the individual types but we want to keep this all together during
// prototyping.

// fromProto converts from a protobuf Resource to a Resource interface.

// NOTE(stevvooe): An alternative model that supports inline declaration.
// Convenient for unit testing where inline declarations may be desirable but
// creates an awkward API for the standard use case.

// type ResourceKind int

// const (
// 	ResourceRegularFile = iota + 1
// 	ResourceDirectory
// 	ResourceSymLink
// 	Resource
// )

// type Resource struct {
// 	Kind         ResourceKind
// 	Paths        []string
// 	Mode         os.FileMode
// 	UID          string
// 	GID          string
// 	Size         int64
// 	Digests      []digest.Digest
// 	Target       string
// 	Major, Minor int
// 	XAttrs       map[string][]byte
// }

// type RegularFile struct {
// 	Paths   []string
//  Size 	int64
// 	Digests []digest.Digest
// 	Perm    os.FileMode // os.ModePerm + sticky, setuid, setgid
// }
