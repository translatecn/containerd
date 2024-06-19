package util

import (
	"demo/over/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func AlphaReqToV1Req(
	alphar protoreflect.ProtoMessage,
	v1r interface{ Unmarshal(_ []byte) error },
) error {
	p, err := proto.Marshal(alphar)
	if err != nil {
		return err
	}

	if err = v1r.Unmarshal(p); err != nil {
		return err
	}
	return nil
}

func V1RespToAlphaResp(
	v1res interface{ Marshal() ([]byte, error) },
	alphares protoreflect.ProtoMessage,
) error {
	p, err := v1res.Marshal()
	if err != nil {
		return err
	}

	if err = proto.Unmarshal(p, alphares); err != nil {
		return err
	}
	return nil
}
