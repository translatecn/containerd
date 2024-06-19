package runtime

// TaskMonitor provides an interface for monitoring of containers within containerd
type TaskMonitor interface {
	Monitor(task Task, labels map[string]string) error
	// Stop stops and removes the provided container from the monitor
	Stop(task Task) error
}

// NewMultiTaskMonitor returns a new TaskMonitor broadcasting to the provided monitors

// NewNoopMonitor is a task monitor that does nothing
func NewNoopMonitor() TaskMonitor {
	return &noopTaskMonitor{}
}

type noopTaskMonitor struct {
}

func (mm *noopTaskMonitor) Monitor(c Task, labels map[string]string) error {
	return nil
}

func (mm *noopTaskMonitor) Stop(c Task) error {
	return nil
}
