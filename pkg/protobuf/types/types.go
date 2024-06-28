// Package types provides convinient aliases that make google.golang.org/protobuf migration easier.
package types

import (
	"google.golang.org/genproto/protobuf/field_mask"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Empty = emptypb.Empty
type Any = anypb.Any
type FieldMask = field_mask.FieldMask
