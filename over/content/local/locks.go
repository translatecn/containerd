package local

import (
	"fmt"
	"sync"
	"time"

	"demo/over/errdefs"
)

// Handles locking references

type lock struct {
	since time.Time
}

var (
	// locks lets us lock in process
	locks   = make(map[string]*lock)
	locksMu sync.Mutex
)

func tryLock(ref string) error {
	locksMu.Lock()
	defer locksMu.Unlock()

	if v, ok := locks[ref]; ok {
		// Returning the duration may help developers distinguish dead locks (long duration) from
		// lock contentions (short duration).
		now := time.Now()
		return fmt.Errorf(
			"ref %s locked for %s (since %s): %w", ref, now.Sub(v.since), v.since,
			errdefs.ErrUnavailable,
		)
	}

	locks[ref] = &lock{time.Now()}
	return nil
}

func unlock(ref string) {
	locksMu.Lock()
	defer locksMu.Unlock()

	delete(locks, ref)
}
