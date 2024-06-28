package stats

import "time"

// ContainerStats contains the information about container stats.
type ContainerStats struct {
	// Timestamp of when stats were collected
	Timestamp time.Time
	// Cumulative CPU usage (sum across all cores) since object creation.
	UsageCoreNanoSeconds uint64
}
