package content

import (
	"demo/over/filters"
	"strings"
)

// AdaptInfo returns `filters.Adaptor` that handles `content.Info`.
func AdaptInfo(info Info) filters.Adaptor {
	return filters.AdapterFunc(func(fieldpath []string) (string, bool) {
		if len(fieldpath) == 0 {
			return "", false
		}

		switch fieldpath[0] {
		case "digest":
			return info.Digest.String(), true
		case "size":
			// TODO: support size based filtering
		case "labels":
			return checkMap(fieldpath[1:], info.Labels)
		}

		return "", false
	})
}

func checkMap(fieldpath []string, m map[string]string) (string, bool) {
	if len(m) == 0 {
		return "", false
	}

	value, ok := m[strings.Join(fieldpath, ".")]
	return value, ok
}
