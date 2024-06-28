package encryption

import (
	"reflect"

	"demo/pkg/diff"
	"github.com/gogo/protobuf/types"
)

var processorPayloadsUseGogo bool

func init() {
	var c = &diff.ApplyConfig{}
	var pbany *types.Any

	pp := reflect.TypeOf(c.ProcessorPayloads)
	processorPayloadsUseGogo = pp.Elem() == reflect.TypeOf(pbany)
}

func clearProcessorPayloads(c *diff.ApplyConfig) {
	var empty = reflect.MakeMap(reflect.TypeOf(c.ProcessorPayloads))
	reflect.ValueOf(&c.ProcessorPayloads).Elem().Set(empty)
}

func setProcessorPayload(c *diff.ApplyConfig, id string, value pbAny) {
	if c.ProcessorPayloads == nil {
		clearProcessorPayloads(c)
	}

	var v reflect.Value
	if processorPayloadsUseGogo {
		v = reflect.ValueOf(fromAny(value))
	} else {
		v = reflect.ValueOf(value)
	}
	reflect.ValueOf(c.ProcessorPayloads).SetMapIndex(reflect.ValueOf(id), v)
}
