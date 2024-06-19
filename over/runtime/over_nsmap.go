package runtime

import (
	"context"
	"demo/over/namespaces"
	"fmt"
	"sync"

	"demo/over/errdefs"
)

type object interface {
	ID() string
}

// NSMap extends Map type with a notion of namespaces passed via Context.
type NSMap[T object] struct {
	mu      sync.Mutex
	objects map[string]map[string]T // {ns}:map[string]T
}

// NewNSMap returns a new NSMap
func NewNSMap[T object]() *NSMap[T] {
	return &NSMap[T]{
		objects: make(map[string]map[string]T),
	}
}

// Get a task
func (m *NSMap[T]) Get(ctx context.Context, id string) (T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	namespace, err := namespaces.NamespaceRequired(ctx)
	var t T
	if err != nil {
		return t, err
	}
	tasks, ok := m.objects[namespace]
	if !ok {
		return t, errdefs.ErrNotFound
	}
	t, ok = tasks[id]
	if !ok {
		return t, errdefs.ErrNotFound
	}
	return t, nil
}

// GetAll objects under a namespace
func (m *NSMap[T]) GetAll(ctx context.Context, noNS bool) ([]T, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var o []T
	if noNS {
		for ns := range m.objects {
			for _, t := range m.objects[ns] {
				o = append(o, t)
			}
		}
		return o, nil
	}
	namespace, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return nil, err
	}
	tasks, ok := m.objects[namespace]
	if !ok {
		return o, nil
	}
	for _, t := range tasks {
		o = append(o, t)
	}
	return o, nil
}

func (m *NSMap[T]) Add(ctx context.Context, t T) error {
	namespace, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return err
	}
	return m.AddWithNamespace(namespace, t)
}

// AddWithNamespace adds a task with the provided namespace
func (m *NSMap[T]) AddWithNamespace(namespace string, t T) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := t.ID()
	if _, ok := m.objects[namespace]; !ok {
		m.objects[namespace] = make(map[string]T)
	}
	if _, ok := m.objects[namespace][id]; ok {
		return fmt.Errorf("%s: %w", id, errdefs.ErrAlreadyExists)
	}
	m.objects[namespace][id] = t
	return nil
}

// Delete a task
func (m *NSMap[T]) Delete(ctx context.Context, id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	namespace, err := namespaces.NamespaceRequired(ctx)
	if err != nil {
		return
	}
	tasks, ok := m.objects[namespace]
	if ok {
		delete(tasks, id)
	}
}

func (m *NSMap[T]) IsEmpty() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for ns := range m.objects {
		if len(m.objects[ns]) > 0 {
			return false
		}
	}

	return true
}
