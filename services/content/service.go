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
	over_plugin2 "demo/over/plugin"
	"errors"

	"demo/content"
	"demo/services"
	"demo/services/content/contentserver"
)

func init() {
	over_plugin2.Register(&over_plugin2.Registration{
		Type: over_plugin2.GRPCPlugin,
		ID:   "content",
		Requires: []over_plugin2.Type{
			over_plugin2.ServicePlugin,
		},
		InitFn: func(ic *over_plugin2.InitContext) (interface{}, error) {
			plugins, err := ic.GetByType(over_plugin2.ServicePlugin)
			if err != nil {
				return nil, err
			}
			p, ok := plugins[services.ContentService]
			if !ok {
				return nil, errors.New("content store service not found")
			}
			cs, err := p.Instance()
			if err != nil {
				return nil, err
			}
			return contentserver.New(cs.(content.Store)), nil
		},
	})
}
