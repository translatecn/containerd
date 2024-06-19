// Package gc experiments with providing central gc tooling to ensure
// deterministic resource removal within containerd.
//
// For now, we just have a single exported implementation that can be used
// under certain use cases.
package gc

import (
	"time"
)

// ResourceType represents type of resource at a node
type ResourceType uint8

// ResourceMax represents the max resource.
// Upper bits are stripped out during the mark phase, allowing the upper 3 bits
// to be used by the caller reference function.
const ResourceMax = ResourceType(0x1F)

// Node presents a resource which has a type and key,
// this node can be used to lookup other nodes.
type Node struct {
	Type      ResourceType
	Namespace string
	Key       string
}

// Stats about a garbage collection run
type Stats interface {
	Elapsed() time.Duration
}

// Tricolor implements basic, single-thread tri-color GC. Given the roots, the
// complete set and a refs function, this function returns a map of all
// reachable objects.
//
// Correct usage requires that the caller not allow the arguments to change
// until the result is used to delete objects in the system.
//
// It will allocate memory proportional to the size of the reachable set.
//
// We can probably use this to inform a design for incremental GC by injecting
// callbacks to the set modification algorithms.
//
// https://en.wikipedia.org/wiki/Tracing_garbage_collection#Tri-color_marking
func Tricolor(roots []Node, refs func(ref Node) ([]Node, error)) (map[Node]struct{}, error) {
	var (
		grays     []Node                // maintain a gray "stack"
		seen      = map[Node]struct{}{} // or not "white", basically "seen"
		reachable = map[Node]struct{}{} // or "black", in tri-color parlance
	)

	grays = append(grays, roots...)

	for len(grays) > 0 {
		// Pick any gray object
		id := grays[len(grays)-1] // effectively "depth first" because first element
		grays = grays[:len(grays)-1]
		seen[id] = struct{}{} // post-mark this as not-white
		rs, err := refs(id)
		if err != nil {
			return nil, err
		}

		// mark all the referenced objects as gray
		for _, target := range rs {
			if _, ok := seen[target]; !ok {
				grays = append(grays, target)
			}
		}

		// strip bits above max resource type
		id.Type = id.Type & ResourceMax
		// mark as black when done
		reachable[id] = struct{}{}
	}

	return reachable, nil
}
