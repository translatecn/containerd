package api

import (
	"sort"

	rspec "github.com/opencontainers/runtime-spec/specs-go"
)

const (
	// SELinuxRelabel is a Mount pseudo-option to request relabeling.
	SELinuxRelabel = "relabel"
)

// FromOCIMounts returns a Mount slice for an OCI runtime Spec.
func FromOCIMounts(o []rspec.Mount) []*Mount {
	var mounts []*Mount
	for _, m := range o {
		mounts = append(mounts, &Mount{
			Destination: m.Destination,
			Type:        m.Type,
			Source:      m.Source,
			Options:     DupStringSlice(m.Options),
		})
	}
	return mounts
}

// ToOCI returns a Mount for an OCI runtime Spec.
func (m *Mount) ToOCI(propagationQuery *string) rspec.Mount {
	o := rspec.Mount{
		Destination: m.Destination,
		Type:        m.Type,
		Source:      m.Source,
	}
	for _, opt := range m.Options {
		o.Options = append(o.Options, opt)
		if propagationQuery != nil && (opt == "rprivate" || opt == "rshared" || opt == "rslave") {
			*propagationQuery = opt
		}
	}
	return o
}

// Cmp returns true if the mounts are equal.
func (m *Mount) Cmp(v *Mount) bool {
	if v == nil {
		return false
	}
	if m.Destination != v.Destination || m.Type != v.Type || m.Source != v.Source ||
		len(m.Options) != len(v.Options) {
		return false
	}

	mOpts := make([]string, len(m.Options))
	vOpts := make([]string, len(m.Options))
	sort.Strings(mOpts)
	sort.Strings(vOpts)

	for i, o := range mOpts {
		if vOpts[i] != o {
			return false
		}
	}

	return true
}

// IsMarkedForRemoval checks if a Mount is marked for removal.
func (m *Mount) IsMarkedForRemoval() (string, bool) {
	key, marked := IsMarkedForRemoval(m.Destination)
	return key, marked
}
