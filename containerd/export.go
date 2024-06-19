package containerd

import (
	"context"
	"io"

	"demo/over/images/archive"
)

// Export exports images to a Tar stream.
// The tar archive is in OCI format with a Docker compatible manifest
// when a single target platform is given.
func (c *Client) Export(ctx context.Context, w io.Writer, opts ...archive.ExportOpt) error {
	return archive.Export(ctx, c.ContentStore(), w, opts...)
}
