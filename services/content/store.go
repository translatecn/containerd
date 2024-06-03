/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package content

import (
	"context"
	over_plugin2 "demo/over/plugin"

	"demo/content"
	eventstypes "demo/pkg/api/events"
	"demo/pkg/events"
	"demo/pkg/metadata"
	"demo/services"
	digest "github.com/opencontainers/go-digest"
)

// store wraps content.Store with proper event published.
type store struct {
	content.Store
	publisher events.Publisher
}

func init() {
	over_plugin2.Register(&over_plugin2.Registration{
		Type: over_plugin2.ServicePlugin,
		ID:   services.ContentService,
		Requires: []over_plugin2.Type{
			over_plugin2.EventPlugin,
			over_plugin2.MetadataPlugin,
		},
		InitFn: func(ic *over_plugin2.InitContext) (interface{}, error) {
			m, err := ic.Get(over_plugin2.MetadataPlugin)
			if err != nil {
				return nil, err
			}
			ep, err := ic.Get(over_plugin2.EventPlugin)
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
