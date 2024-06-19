package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// TimeFormat is [time.RFC3339Nano] with nanoseconds padded using
// zeros to ensure the formatted time is always the same number of
// characters.
// Based on RFC3339NanoFixed from github.com/containerd/log
const TimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

// DurationFormat formats a [time.Duration] log entry.
//
// A nil value signals an error with the formatting.
type DurationFormat func(time.Duration) interface{}

// FormatIO formats net.Conn and other types that have an `Addr()` or `Name()`.
//
// See FormatEnabled for more information.

// Format formats an object into a JSON string, without any indendtation or
// HTML escapes.
// Context is used to output a log waring if the conversion fails.
//
// This is intended primarily for `trace.StringAttribute()`

func encode(v interface{}) ([]byte, error) {
	return encodeBuffer(&bytes.Buffer{}, v)
}

func encodeBuffer(buf *bytes.Buffer, v interface{}) ([]byte, error) {
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "")

	if err := enc.Encode(v); err != nil {
		err = fmt.Errorf("could not marshall %T to JSON for logging: %w", v, err)
		return nil, err
	}

	// encoder.Encode appends a newline to the end
	return bytes.TrimSpace(buf.Bytes()), nil
}
