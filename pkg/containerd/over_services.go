package containerd

import (
	"demo/pkg/api/services/diff/v1"
	"demo/pkg/api/services/tasks/v1"
	"demo/pkg/leases"
	"demo/pkg/namespaces"
	"demo/pkg/plugin"
	srv "demo/pkg/plugins"
	"demo/pkg/plugins/containerd/service/introspection"
	sandbox2 "demo/pkg/sandbox"
	"demo/pkg/snapshots"
	"fmt"

	containersapi "demo/pkg/api/services/containers/v1"
	imagesapi "demo/pkg/api/services/images/v1"
	introspectionapi "demo/pkg/api/services/introspection/v1"
	namespacesapi "demo/pkg/api/services/namespaces/v1"
	"demo/pkg/containers"
	"demo/pkg/content"
	"demo/pkg/images"
)

type services struct {
	contentStore         content.Store
	imageStore           images.Store
	containerStore       containers.Store
	namespaceStore       namespaces.Store
	snapshotters         map[string]snapshots.Snapshotter
	taskService          tasks.TasksClient
	diffService          DiffService
	eventService         EventService
	leasesService        leases.Manager
	introspectionService introspection.Service
	sandboxStore         sandbox2.Store
	sandboxController    sandbox2.Controller
}

// ServicesOpt allows callers to set options on the services
type ServicesOpt func(c *services)

// WithContentStore sets the content store.
func WithContentStore(contentStore content.Store) ServicesOpt {
	return func(s *services) {
		s.contentStore = contentStore
	}
}

// WithImageClient sets the image service to use using an images client.
func WithImageClient(imageService imagesapi.ImagesClient) ServicesOpt {
	return func(s *services) {
		s.imageStore = NewImageStoreFromClient(imageService)
	}
}

// WithImageStore sets the image store.

// WithSnapshotters sets the snapshotters.
func WithSnapshotters(snapshotters map[string]snapshots.Snapshotter) ServicesOpt {
	return func(s *services) {
		s.snapshotters = make(map[string]snapshots.Snapshotter)
		for n, sn := range snapshotters {
			s.snapshotters[n] = sn
		}
	}
}

// WithContainerClient sets the container service to use using a containers client.
func WithContainerClient(containerService containersapi.ContainersClient) ServicesOpt {
	return func(s *services) {
		s.containerStore = NewRemoteContainerStore(containerService)
	}
}

// WithTaskClient sets the task service to use from a tasks client.
func WithTaskClient(taskService tasks.TasksClient) ServicesOpt {
	return func(s *services) {
		s.taskService = taskService
	}
}

// WithDiffClient sets the diff service to use from a diff client.
func WithDiffClient(diffService diff.DiffClient) ServicesOpt {
	return func(s *services) {
		s.diffService = NewDiffServiceFromClient(diffService)
	}
}

// WithEventService sets the event service.
func WithEventService(eventService EventService) ServicesOpt {
	return func(s *services) {
		s.eventService = eventService
	}
}

// WithNamespaceClient sets the namespace service using a namespaces client.
func WithNamespaceClient(namespaceService namespacesapi.NamespacesClient) ServicesOpt {
	return func(s *services) {
		s.namespaceStore = NewNamespaceStoreFromClient(namespaceService)
	}
}

// WithLeasesService sets the lease service.
func WithLeasesService(leasesService leases.Manager) ServicesOpt {
	return func(s *services) {
		s.leasesService = leasesService
	}
}

// WithIntrospectionClient sets the introspection service using an introspection client.
func WithIntrospectionClient(in introspectionapi.IntrospectionClient) ServicesOpt {
	return func(s *services) {
		s.introspectionService = introspection.NewIntrospectionServiceFromClient(in)
	}
}

// WithSandboxStore sets the sandbox store.
func WithSandboxStore(client sandbox2.Store) ServicesOpt {
	return func(s *services) {
		s.sandboxStore = client
	}
}

// WithSandboxController sets the sandbox controller.
func WithSandboxController(client sandbox2.Controller) ServicesOpt {
	return func(s *services) {
		s.sandboxController = client
	}
}

// WithInMemoryServices 将各种插件的实现塞到ic里
func WithInMemoryServices(ic *plugin.InitContext) ClientOpt {
	return func(c *clientOpts) error {
		var opts []ServicesOpt
		for t, fn := range map[plugin.Type]func(interface{}) ServicesOpt{
			plugin.EventPlugin: func(i interface{}) ServicesOpt {
				return WithEventService(i.(EventService))
			},
			plugin.LeasePlugin: func(i interface{}) ServicesOpt {
				return WithLeasesService(i.(leases.Manager))
			},
			plugin.SandboxStorePlugin: func(i interface{}) ServicesOpt {
				return WithSandboxStore(i.(sandbox2.Store))
			},
			plugin.SandboxControllerPlugin: func(i interface{}) ServicesOpt {
				return WithSandboxController(i.(sandbox2.Controller))
			},
		} {
			i, err := ic.Get(t)
			if err != nil {
				return fmt.Errorf("failed to get %q plugin: %w", t, err)
			}
			opts = append(opts, fn(i))
		}

		plugins, err := ic.GetByType(plugin.ServicePlugin)
		if err != nil {
			return fmt.Errorf("failed to get service plugin: %w", err)
		}
		for s, fn := range map[string]func(interface{}) ServicesOpt{
			srv.ContentService: func(s interface{}) ServicesOpt {
				return WithContentStore(s.(content.Store))
			},
			srv.ImagesService: func(s interface{}) ServicesOpt {
				return WithImageClient(s.(imagesapi.ImagesClient))
			},
			srv.SnapshotsService: func(s interface{}) ServicesOpt {
				return WithSnapshotters(s.(map[string]snapshots.Snapshotter))
			},
			srv.ContainersService: func(s interface{}) ServicesOpt {
				return WithContainerClient(s.(containersapi.ContainersClient))
			},
			srv.TasksService: func(s interface{}) ServicesOpt {
				return WithTaskClient(s.(tasks.TasksClient))
			},
			srv.DiffService: func(s interface{}) ServicesOpt {
				return WithDiffClient(s.(diff.DiffClient))
			},
			srv.NamespacesService: func(s interface{}) ServicesOpt {
				return WithNamespaceClient(s.(namespacesapi.NamespacesClient))
			},
			srv.IntrospectionService: func(s interface{}) ServicesOpt {
				return WithIntrospectionClient(s.(introspectionapi.IntrospectionClient))
			},
		} {
			p := plugins[s]
			if p == nil {
				return fmt.Errorf("service %q not found", s)
			}
			i, err := p.Instance()
			if err != nil {
				return fmt.Errorf("failed to get instance of service %q: %w", s, err)
			}
			if i == nil {
				return fmt.Errorf("instance of service %q not found", s)
			}
			opts = append(opts, fn(i))
		}

		c.services = &services{}
		for _, o := range opts {
			o(c.services)
		}
		return nil
	}
}
