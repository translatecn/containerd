package metadata

import (
	"demo/pkg/leases"
	"demo/pkg/sandbox"
	"demo/pkg/snapshots"
	"strings"

	"demo/pkg/containers"
	"demo/pkg/content"
	"demo/pkg/filters"
	"demo/pkg/images"
)

func adaptImage(o interface{}) filters.Adaptor {
	obj := o.(images.Image)
	return filters.AdapterFunc(func(fieldpath []string) (string, bool) {
		if len(fieldpath) == 0 {
			return "", false
		}

		switch fieldpath[0] {
		case "name":
			return obj.Name, len(obj.Name) > 0
		case "target":
			if len(fieldpath) < 2 {
				return "", false
			}

			switch fieldpath[1] {
			case "digest":
				return obj.Target.Digest.String(), len(obj.Target.Digest) > 0
			case "mediatype":
				return obj.Target.MediaType, len(obj.Target.MediaType) > 0
			}
		case "labels":
			return checkMap(fieldpath[1:], obj.Labels)
			// TODO(stevvooe): Greater/Less than filters would be awesome for
			// size. Let's do it!
		case "annotations":
			return checkMap(fieldpath[1:], obj.Target.Annotations)
		}

		return "", false
	})
}
func adaptContainer(o interface{}) filters.Adaptor {
	obj := o.(containers.Container)
	return filters.AdapterFunc(func(fieldpath []string) (string, bool) {
		if len(fieldpath) == 0 {
			return "", false
		}

		switch fieldpath[0] {
		case "id":
			return obj.ID, len(obj.ID) > 0
		case "runtime":
			if len(fieldpath) <= 1 {
				return "", false
			}

			switch fieldpath[1] {
			case "name":
				return obj.Runtime.Name, len(obj.Runtime.Name) > 0
			default:
				return "", false
			}
		case "image":
			return obj.Image, len(obj.Image) > 0
		case "labels":
			return checkMap(fieldpath[1:], obj.Labels)
		}

		return "", false
	})
}

func adaptContentStatus(status content.Status) filters.Adaptor {
	return filters.AdapterFunc(func(fieldpath []string) (string, bool) {
		if len(fieldpath) == 0 {
			return "", false
		}
		switch fieldpath[0] {
		case "ref":
			return status.Ref, true
		}

		return "", false
	})
}

func adaptLease(lease leases.Lease) filters.Adaptor {
	return filters.AdapterFunc(func(fieldpath []string) (string, bool) {
		if len(fieldpath) == 0 {
			return "", false
		}

		switch fieldpath[0] {
		case "id":
			return lease.ID, len(lease.ID) > 0
		case "labels":
			return checkMap(fieldpath[1:], lease.Labels)
		}

		return "", false
	})
}

func adaptSnapshot(info snapshots.Info) filters.Adaptor {
	return filters.AdapterFunc(func(fieldpath []string) (string, bool) {
		if len(fieldpath) == 0 {
			return "", false
		}

		switch fieldpath[0] {
		case "kind":
			switch info.Kind {
			case snapshots.KindActive:
				return "active", true
			case snapshots.KindView:
				return "view", true
			case snapshots.KindCommitted:
				return "committed", true
			}
		case "name":
			return info.Name, true
		case "parent":
			return info.Parent, true
		case "labels":
			return checkMap(fieldpath[1:], info.Labels)
		}

		return "", false
	})
}

func adaptSandbox(instance *sandbox.Sandbox) filters.Adaptor {
	return filters.AdapterFunc(func(fieldpath []string) (string, bool) {
		if len(fieldpath) == 0 {
			return "", false
		}

		switch fieldpath[0] {
		case "id":
			return instance.ID, true
		case "labels":
			return checkMap(fieldpath[1:], instance.Labels)
		default:
			return "", false
		}
	})
}

func checkMap(fieldpath []string, m map[string]string) (string, bool) {
	if len(m) == 0 {
		return "", false
	}

	value, ok := m[strings.Join(fieldpath, ".")]
	return value, ok
}