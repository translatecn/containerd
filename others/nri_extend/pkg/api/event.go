package api

import (
	"fmt"
)

const (
	// ValidEvents is the event mask of all valid events.
	ValidEvents = EventMask((1 << (Event_LAST - 1)) - 1)
)

// nolint
type (
	// Define *Request/*Response type aliases for *Event/Empty pairs.

	StateChangeResponse         = Empty
	RunPodSandboxRequest        = StateChangeEvent
	RunPodSandboxResponse       = Empty
	StopPodSandboxRequest       = StateChangeEvent
	StopPodSandboxResponse      = Empty
	RemovePodSandboxRequest     = StateChangeEvent
	RemovePodSandboxResponse    = Empty
	StartContainerRequest       = StateChangeEvent
	StartContainerResponse      = Empty
	RemoveContainerRequest      = StateChangeEvent
	RemoveContainerResponse     = Empty
	PostCreateContainerRequest  = StateChangeEvent
	PostCreateContainerResponse = Empty
	PostStartContainerRequest   = StateChangeEvent
	PostStartContainerResponse  = Empty
	PostUpdateContainerRequest  = StateChangeEvent
	PostUpdateContainerResponse = Empty

	ShutdownRequest  = Empty
	ShutdownResponse = Empty
)

// EventMask corresponds to a set of enumerated Events.
type EventMask int32

// ParseEventMask parses a string representation into an EventMask.

// PrettyString returns a human-readable string representation of an EventMask.
func (m *EventMask) PrettyString() string {
	names := map[Event]string{
		Event_RUN_POD_SANDBOX:       "RunPodSandbox",
		Event_STOP_POD_SANDBOX:      "StopPodSandbox",
		Event_REMOVE_POD_SANDBOX:    "RemovePodSandbox",
		Event_CREATE_CONTAINER:      "CreateContainer",
		Event_POST_CREATE_CONTAINER: "PostCreateContainer",
		Event_START_CONTAINER:       "StartContainer",
		Event_POST_START_CONTAINER:  "PostStartContainer",
		Event_UPDATE_CONTAINER:      "UpdateContainer",
		Event_POST_UPDATE_CONTAINER: "PostUpdateContainer",
		Event_STOP_CONTAINER:        "StopContainer",
		Event_REMOVE_CONTAINER:      "RemoveContainer",
	}

	mask := *m
	events, sep := "", ""

	for bit := Event_UNKNOWN + 1; bit <= Event_LAST; bit++ {
		if mask.IsSet(bit) {
			events += sep + names[bit]
			sep = ","
			mask.Clear(bit)
		}
	}

	if mask != 0 {
		events += sep + fmt.Sprintf("unknown(0x%x)", mask)
	}

	return events
}

// Set sets the given Events in the mask.
func (m *EventMask) Set(events ...Event) *EventMask {
	for _, e := range events {
		*m |= (1 << (e - 1))
	}
	return m
}

// Clear clears the given Events in the mask.
func (m *EventMask) Clear(events ...Event) *EventMask {
	for _, e := range events {
		*m &^= (1 << (e - 1))
	}
	return m
}

// IsSet check if the given Event is set in the mask.
func (m *EventMask) IsSet(e Event) bool {
	return *m&(1<<(e-1)) != 0
}
