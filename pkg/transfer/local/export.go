package local

import (
	"context"

	"demo/pkg/images"
	"demo/pkg/transfer"
)

func (ts *localTransferService) exportStream(ctx context.Context, ig transfer.ImageGetter, is transfer.ImageExporter, tops *transfer.Config) error {
	ctx, done, err := ts.withLease(ctx)
	if err != nil {
		return err
	}
	defer done(ctx)

	if tops.Progress != nil {
		tops.Progress(transfer.Progress{
			Event: "Exporting",
		})
	}

	var imgs []images.Image
	if il, ok := ig.(transfer.ImageLookup); ok {
		imgs, err = il.Lookup(ctx, ts.images)
		if err != nil {
			return err
		}
	} else {
		img, err := ig.Get(ctx, ts.images)
		if err != nil {
			return err
		}
		imgs = append(imgs, img)
	}

	err = is.Export(ctx, ts.content, imgs)
	if err != nil {
		return err
	}

	if tops.Progress != nil {
		tops.Progress(transfer.Progress{
			Event: "Completed export",
		})
	}
	return nil
}
