// Package epoch provides SOURCE_DATE_EPOCH utilities.
package epoch

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// SourceDateEpochEnv is the SOURCE_DATE_EPOCH env var.
// See https://reproducible-builds.org/docs/source-date-epoch/
const SourceDateEpochEnv = "SOURCE_DATE_EPOCH"

// SourceDateEpoch returns the SOURCE_DATE_EPOCH env var as *time.Time.
// If the env var is not set, SourceDateEpoch returns nil without an error.
func SourceDateEpoch() (*time.Time, error) {
	v, ok := os.LookupEnv(SourceDateEpochEnv)
	if !ok || v == "" {
		return nil, nil // not an error
	}
	i64, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid %s value %q: %w", SourceDateEpochEnv, v, err)
	}
	unix := time.Unix(i64, 0).UTC()
	return &unix, nil
}
