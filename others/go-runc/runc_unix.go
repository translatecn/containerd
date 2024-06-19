package runc

import (
	"golang.org/x/sys/unix"
)

// Runc is the client to the runc cli
type Runc struct {
	//If command is empty, DefaultCommand is used
	Command       string
	Root          string
	Debug         bool
	Log           string
	LogFormat     Format
	PdeathSignal  unix.Signal
	Setpgid       bool
	Criu          string
	SystemdCgroup bool
	Rootless      *bool // nil stands for "auto"
}
