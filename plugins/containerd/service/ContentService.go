package service

import (
	"context"
	"demo/over/metadata"
	"demo/over/plugin"
	"demo/plugins"

	eventstypes "demo/over/api/events"
	"demo/over/content"
	"demo/over/events"
	digest "github.com/opencontainers/go-digest"
)

// store wraps content.Store with proper event published.
type store struct {
	content.Store
	publisher events.Publisher
}

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.ServicePlugin,
		ID:   plugins.ContentService,
		Requires: []plugin.Type{
			plugin.EventPlugin,
			plugin.MetadataPlugin,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			m, err := ic.Get(plugin.MetadataPlugin)
			if err != nil {
				return nil, err
			}
			ep, err := ic.Get(plugin.EventPlugin)
			if err != nil {
				return nil, err
			}

			s, err := newContentStore(m.(*metadata.DB).ContentStore(), ep.(events.Publisher))
			return s, err
		},
	})
}

func newContentStore(cs content.Store, publisher events.Publisher) (content.Store, error) {
	return &store{
		Store:     cs,
		publisher: publisher,
	}, nil
}

func (s *store) Delete(ctx context.Context, dgst digest.Digest) error {
	if err := s.Store.Delete(ctx, dgst); err != nil {
		return err
	}
	// TODO: Consider whether we should return error here.
	return s.publisher.Publish(ctx, "/content/delete", &eventstypes.ContentDelete{
		Digest: dgst.String(),
	})
}
