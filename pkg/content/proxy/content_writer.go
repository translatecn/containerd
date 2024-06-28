package proxy

import (
	"context"
	"demo/pkg/protobuf"
	"fmt"
	"io"

	contentapi "demo/pkg/api/services/content/v1"
	"demo/pkg/content"
	"demo/pkg/errdefs"
	digest "github.com/opencontainers/go-digest"
)

type remoteWriter struct {
	ref    string
	client contentapi.Content_WriteClient
	offset int64
	digest digest.Digest
}

// send performs a synchronous req-resp cycle on the client.
func (rw *remoteWriter) send(req *contentapi.WriteContentRequest) (*contentapi.WriteContentResponse, error) {
	if err := rw.client.Send(req); err != nil {
		return nil, err
	}

	resp, err := rw.client.Recv()

	if err == nil {
		// try to keep these in sync
		if resp.Digest != "" {
			rw.digest = digest.Digest(resp.Digest)
		}
	}

	return resp, err
}

func (rw *remoteWriter) Status() (content.Status, error) {
	resp, err := rw.send(&contentapi.WriteContentRequest{
		Action: contentapi.WriteAction_STAT,
	})
	if err != nil {
		return content.Status{}, fmt.Errorf("error getting writer status: %w", errdefs.FromGRPC(err))
	}

	return content.Status{
		Ref:       rw.ref,
		Offset:    resp.Offset,
		Total:     resp.Total,
		StartedAt: protobuf.FromTimestamp(resp.StartedAt),
		UpdatedAt: protobuf.FromTimestamp(resp.UpdatedAt),
	}, nil
}

func (rw *remoteWriter) Digest() digest.Digest {
	return rw.digest
}

func (rw *remoteWriter) Write(p []byte) (n int, err error) {
	offset := rw.offset

	resp, err := rw.send(&contentapi.WriteContentRequest{
		Action: contentapi.WriteAction_WRITE,
		Offset: offset,
		Data:   p,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to send write: %w", errdefs.FromGRPC(err))
	}

	n = int(resp.Offset - offset)
	if n < len(p) {
		err = io.ErrShortWrite
	}

	rw.offset += int64(n)
	if resp.Digest != "" {
		rw.digest = digest.Digest(resp.Digest)
	}
	return
}

func (rw *remoteWriter) Commit(ctx context.Context, size int64, expected digest.Digest, opts ...content.Opt) (err error) {
	defer func() {
		err1 := rw.Close()
		if err == nil {
			err = err1
		}
	}()

	var base content.Info
	for _, opt := range opts {
		if err := opt(&base); err != nil {
			return err
		}
	}
	resp, err := rw.send(&contentapi.WriteContentRequest{
		Action:   contentapi.WriteAction_COMMIT,
		Total:    size,
		Offset:   rw.offset,
		Expected: expected.String(),
		Labels:   base.Labels,
	})
	if err != nil {
		return fmt.Errorf("commit failed: %w", errdefs.FromGRPC(err))
	}

	if size != 0 && resp.Offset != size {
		return fmt.Errorf("unexpected size: %v != %v", resp.Offset, size)
	}

	actual := digest.Digest(resp.Digest)
	if expected != "" && actual != expected {
		return fmt.Errorf("unexpected digest: %v != %v", resp.Digest, expected)
	}

	rw.digest = actual
	rw.offset = resp.Offset
	return nil
}

func (rw *remoteWriter) Truncate(size int64) error {
	// This truncation won't actually be validated until a write is issued.
	rw.offset = size
	return nil
}

func (rw *remoteWriter) Close() error {
	return rw.client.CloseSend()
}
