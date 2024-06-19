package streaming

import (
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// NewErrorStreamingDisabled creates an error for disabled streaming method.

// NewErrorTooManyInFlight creates an error for exceeding the maximum number of in-flight requests.
func NewErrorTooManyInFlight() error {
	return grpcstatus.Error(codes.ResourceExhausted, "maximum number of in-flight requests exceeded")
}

// WriteError translates a CRI streaming error into an appropriate HTTP response.
