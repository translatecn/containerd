package local

import (
	"context"

	"demo/over/transfer"
)

func (ts *localTransferService) tag(ctx context.Context, ig transfer.ImageGetter, is transfer.ImageStorer, tops *transfer.Config) error {
	ctx, done, err := ts.withLease(ctx)
	if err != nil {
		return err
	}
	defer done(ctx)

	img, err := ig.Get(ctx, ts.images)
	if err != nil {
		return err
	}

	_, err = is.Store(ctx, img.Target, ts.images)
	return err
}
