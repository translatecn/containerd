package protobuf

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Once we migrate off from gogo/protobuf, we can use the function below, which don't return any errors.
// https://github.com/protocolbuffers/protobuf-go/blob/v1.28.0/types/known/timestamppb/timestamp.pb.go#L200-L208

// ToTimestamp creates protobuf's Timestamp from time.Time.
func ToTimestamp(from time.Time) *timestamppb.Timestamp {
	return timestamppb.New(from)
}

// FromTimestamp creates time.Time from protobuf's Timestamp.
func FromTimestamp(from *timestamppb.Timestamp) time.Time {
	return from.AsTime()
}
