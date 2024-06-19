package oc

import (
	"go.opencensus.io/trace"
)

var _ = trace.AlwaysSample()

// SetSpanStatus sets `span.SetStatus` to the proper status depending on `err`. If
// `err` is `nil` assumes `trace.StatusCodeOk`.

// StartSpan wraps "go.opencensus.io/trace".StartSpan, but, if the span is sampling,
// adds a log entry to the context that points to the newly created span.

// StartSpanWithRemoteParent wraps "go.opencensus.io/trace".StartSpanWithRemoteParent.
//
// See StartSpan for more information.

var _ = trace.WithSpanKind(trace.SpanKindServer)
var _ = trace.WithSpanKind(trace.SpanKindClient)

func spanKindToString(sk int) string {
	switch sk {
	case trace.SpanKindClient:
		return "client"
	case trace.SpanKindServer:
		return "server"
	default:
		return ""
	}
}
