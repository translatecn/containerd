package plugins

import (
	"demo/over/typeurl/v2"
	"fmt"
	"reflect"
	"sync"

	"demo/over/errdefs"
)

var register = struct {
	sync.RWMutex
	r map[string]reflect.Type
}{}

func Register(apiObject, transferObject interface{}) {
	url, err := typeurl.TypeURL(apiObject)
	if err != nil {
		panic(err)
	}
	// Lock
	register.Lock()
	defer register.Unlock()
	if register.r == nil {
		register.r = map[string]reflect.Type{}
	}
	if _, ok := register.r[url]; ok {
		panic(fmt.Sprintf("url already registered: %v", url))
	}
	t := reflect.TypeOf(transferObject)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	register.r[url] = t
}

func ResolveType(any typeurl.Any) (interface{}, error) {
	register.RLock()
	defer register.RUnlock()
	if register.r != nil {
		if t, ok := register.r[any.GetTypeUrl()]; ok {
			return reflect.New(t).Interface(), nil
		}
	}
	return nil, fmt.Errorf("%v not registered: %w", any.GetTypeUrl(), errdefs.ErrNotFound)
}
