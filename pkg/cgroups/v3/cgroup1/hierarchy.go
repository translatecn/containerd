package cgroup1

// Hierarchy enables both unified and split hierarchy for cgroups
type Hierarchy func() ([]Subsystem, error)
