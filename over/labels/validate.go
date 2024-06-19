package labels

import (
	"fmt"

	"demo/over/errdefs"
)

const (
	maxSize = 4096
	// maximum length of key portion of error message if len of key + len of value > maxSize
	keyMaxLen = 64
)

// Validate a label's key and value are under 4096 bytes
func Validate(k, v string) error {
	total := len(k) + len(v)
	if total > maxSize {
		if len(k) > keyMaxLen {
			k = k[:keyMaxLen]
		}
		return fmt.Errorf("label key and value length (%d bytes) greater than maximum size (%d bytes), key: %s: %w", total, maxSize, k, errdefs.ErrInvalidArgument)
	}
	return nil
}
