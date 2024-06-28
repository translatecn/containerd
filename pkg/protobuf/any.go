package protobuf

import (
	"demo/pkg/typeurl/v2"
	"google.golang.org/protobuf/types/known/anypb"
)

// FromAny converts typeurl.Any to demo/protobuf/types.Any.
func FromAny(from typeurl.Any) *anypb.Any {
	if from == nil {
		return nil
	}

	if pbany, ok := from.(*anypb.Any); ok {
		return pbany
	}

	return &anypb.Any{
		TypeUrl: from.GetTypeUrl(),
		Value:   from.GetValue(),
	}
}

// FromAny converts an arbitrary interface to demo/protobuf/types.Any.
func MarshalAnyToProto(from interface{}) (*anypb.Any, error) {
	any, err := typeurl.MarshalAny(from)
	if err != nil {
		return nil, err
	}
	return FromAny(any), nil
}
