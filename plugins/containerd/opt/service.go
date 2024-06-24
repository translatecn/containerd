package opt

import (
	"demo/over/my_mk"
	"demo/over/plugin"
	"fmt"
	"os"
	"path/filepath"
)

// Config for the opt manager
type Config struct {
	// Path for the opt directory
	Path string `toml:"path"`
}

func init() {
	plugin.Register(&plugin.Registration{
		Type: plugin.InternalPlugin,
		ID:   "opt",
		Config: &Config{
			Path: defaultPath,
		},
		InitFn: func(ic *plugin.InitContext) (interface{}, error) {
			path := ic.Config.(*Config).Path
			ic.Meta.Exports["path"] = path
			bin := filepath.Join(path, "bin")
			if err := my_mk.MkdirAll(bin, 0711); err != nil {
				return nil, err
			}
			if err := os.Setenv("PATH", fmt.Sprintf("%s%c%s", bin, os.PathListSeparator, os.Getenv("PATH"))); err != nil {
				return nil, fmt.Errorf("set binary image directory in path %s: %w", bin, err)
			}

			lib := filepath.Join(path, "lib")
			if err := my_mk.MkdirAll(lib, 0711); err != nil {
				return nil, err
			}
			if err := os.Setenv("LD_LIBRARY_PATH", fmt.Sprintf("%s%c%s", lib, os.PathListSeparator, os.Getenv("LD_LIBRARY_PATH"))); err != nil {
				return nil, fmt.Errorf("set binary lib directory in path %s: %w", lib, err)
			}
			return &manager{}, nil
		},
	})
}

type manager struct {
}
