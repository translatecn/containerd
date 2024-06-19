package util

import (
	"encoding/json"
	"errors"
	"fmt"
)

// DeepCopy makes a deep copy from src into dst.
func DeepCopy(dst interface{}, src interface{}) error {
	if dst == nil {
		return errors.New("dst cannot be nil")
	}
	if src == nil {
		return errors.New("src cannot be nil")
	}
	bytes, err := json.Marshal(src)
	if err != nil {
		return fmt.Errorf("unable to marshal src: %w", err)
	}
	err = json.Unmarshal(bytes, dst)
	if err != nil {
		return fmt.Errorf("unable to unmarshal into dst: %w", err)
	}
	return nil
}
