package encryption

import "github.com/gogo/protobuf/types"

// pbAny takes proto-generated Any type.
// https://developers.google.com/protocol-buffers/docs/proto3#any
type pbAny interface {
	GetTypeUrl() string
	GetValue() []byte
}

func fromAny(from pbAny) *types.Any {
	if from == nil {
		return nil
	}

	pbany, ok := from.(*types.Any)
	if ok {
		return pbany
	}

	return &types.Any{
		TypeUrl: from.GetTypeUrl(),
		Value:   from.GetValue(),
	}
}
