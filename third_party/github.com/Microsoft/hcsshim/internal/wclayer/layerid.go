//go:build windows

package wclayer

import (
	"context"
	"path/filepath"

	"demo/third_party/github.com/Microsoft/hcsshim/internal/oc"
	"github.com/Microsoft/go-winio/pkg/guid"
	"go.opencensus.io/trace"
)

// LayerID returns the layer ID of a layer on disk.
func LayerID(ctx context.Context, path string) (_ guid.GUID, err error) {
	title := "hcsshim::LayerID"
	ctx, span := oc.StartSpan(ctx, title)
	defer span.End()
	defer func() { oc.SetSpanStatus(span, err) }()
	span.AddAttributes(trace.StringAttribute("path", path))

	_, file := filepath.Split(path)
	return NameToGuid(ctx, file)
}
