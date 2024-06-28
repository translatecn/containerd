package registrar

import (
	"fmt"
	"sync"
)

// Registrar stores one-to-one name<->key mappings.
// Names and keys must be unique.
// Registrar is safe for concurrent access.
type Registrar struct {
	lock      sync.Mutex
	nameToKey map[string]string
	keyToName map[string]string
}

// NewRegistrar creates a new Registrar with the empty indexes.
func NewRegistrar() *Registrar {
	return &Registrar{
		nameToKey: make(map[string]string),
		keyToName: make(map[string]string),
	}
}

// Reserve 注册一个name<->键映射，name或key不能为空。
// 保留是幂等的。
// 尝试保留冲突键<->名称映射导致错误。
// 名称<->键保留是全局唯一的。
func (r *Registrar) Reserve(name, key string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if name == "" || key == "" {
		return fmt.Errorf("invalid name %q or key %q", name, key)
	}

	if k, exists := r.nameToKey[name]; exists {
		if k != key {
			return fmt.Errorf("name %q is reserved for %q", name, k)
		}
		return nil
	}

	if n, exists := r.keyToName[key]; exists {
		if n != name {
			return fmt.Errorf("key %q is reserved for %q", key, n)
		}
		return nil
	}

	r.nameToKey[name] = key
	r.keyToName[key] = name
	return nil
}

// ReleaseByName releases the reserved name<->key mapping by name.
// Once released, the name and the key can be reserved again.
func (r *Registrar) ReleaseByName(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	key, exists := r.nameToKey[name]
	if !exists {
		return
	}

	delete(r.nameToKey, name)
	delete(r.keyToName, key)
}

// ReleaseByKey release the reserved name<->key mapping by key.
func (r *Registrar) ReleaseByKey(key string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	name, exists := r.keyToName[key]
	if !exists {
		return
	}

	delete(r.nameToKey, name)
	delete(r.keyToName, key)
}
