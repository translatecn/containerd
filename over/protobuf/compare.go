package protobuf

import (
	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/proto"
)

var Compare = cmp.FilterValues(
	func(x, y interface{}) bool {
		_, xok := x.(proto.Message)
		_, yok := y.(proto.Message)
		return xok && yok
	},
	cmp.Comparer(func(x, y interface{}) bool {
		vx, ok := x.(proto.Message)
		if !ok {
			return false
		}
		vy, ok := y.(proto.Message)
		if !ok {
			return false
		}
		return proto.Equal(vx, vy)
	}),
)
