package log

import (
	"errors"
)

// This package scrubs objects of potentially sensitive information to pass to logging

const _scrubbedReplacement = "<scrubbed>"

var (
	_ = errors.New("encoded object is of unknown type")

	// case sensitive keywords, so "env" is not a substring on "Environment"
	_ = [][]byte{[]byte("env"), []byte("Environment")}

	_ int32
)

// SetScrubbing enables scrubbing

// IsScrubbingEnabled checks if scrubbing is enabled

// ScrubProcessParameters scrubs HCS Create Process requests with config parameters of
// type internal/hcs/schema2.ScrubProcessParameters (aka hcsshema.ScrubProcessParameters)

// ScrubBridgeCreate scrubs requests sent over the bridge of type
// internal/gcs/protocol.containerCreate wrapping an internal/hcsoci.linuxHostedSystem

// ScrubBridgeExecProcess scrubs requests sent over the bridge of type
// internal/gcs/protocol.containerExecuteProcess

// combination `m, ok := m[s]` and `m, ok := m.(genMap)`
