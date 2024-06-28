package ttrpc

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

type codec struct{}

func (c codec) Marshal(msg interface{}) ([]byte, error) {
	switch v := msg.(type) {
	case proto.Message:
		return proto.Marshal(v)
	default:
		return nil, fmt.Errorf("ttrpc: cannot marshal unknown type: %T", msg)
	}
}

func (c codec) Unmarshal(p []byte, msg interface{}) error {
	switch v := msg.(type) {
	case proto.Message:
		return proto.Unmarshal(p, v)
	default:
		return fmt.Errorf("ttrpc: cannot unmarshal into unknown type: %T", msg)
	}
}
