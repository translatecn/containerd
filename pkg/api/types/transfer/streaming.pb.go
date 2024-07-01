// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.20.1
// source: demo/api/types/transfer/streaming.proto

package transfer

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Data struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *Data) Reset() {
	*x = Data{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_containerd_containerd_api_types_transfer_streaming_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Data) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Data) ProtoMessage() {}

func (x *Data) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_containerd_containerd_api_types_transfer_streaming_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Data.ProtoReflect.Descriptor instead.
func (*Data) Descriptor() ([]byte, []int) {
	return file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDescGZIP(), []int{0}
}

func (x *Data) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

type WindowUpdate struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Update int32 `protobuf:"varint,1,opt,name=update,proto3" json:"update,omitempty"`
}

func (x *WindowUpdate) Reset() {
	*x = WindowUpdate{}
	if protoimpl.UnsafeEnabled {
		mi := &file_github_com_containerd_containerd_api_types_transfer_streaming_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WindowUpdate) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WindowUpdate) ProtoMessage() {}

func (x *WindowUpdate) ProtoReflect() protoreflect.Message {
	mi := &file_github_com_containerd_containerd_api_types_transfer_streaming_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WindowUpdate.ProtoReflect.Descriptor instead.
func (*WindowUpdate) Descriptor() ([]byte, []int) {
	return file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDescGZIP(), []int{1}
}

func (x *WindowUpdate) GetUpdate() int32 {
	if x != nil {
		return x.Update
	}
	return 0
}

var File_github_com_containerd_containerd_api_types_transfer_streaming_proto protoreflect.FileDescriptor

var file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDesc = []byte{
	0x0a, 0x43, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6f, 0x6e,
	0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x64, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65,
	0x72, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2f, 0x74, 0x72, 0x61,
	0x6e, 0x73, 0x66, 0x65, 0x72, 0x2f, 0x73, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x69, 0x6e, 0x67, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x19, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72,
	0x64, 0x2e, 0x74, 0x79, 0x70, 0x65, 0x73, 0x2e, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72,
	0x22, 0x1a, 0x0a, 0x04, 0x44, 0x61, 0x74, 0x61, 0x12, 0x12, 0x0a, 0x04, 0x64, 0x61, 0x74, 0x61,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x64, 0x61, 0x74, 0x61, 0x22, 0x26, 0x0a, 0x0c,
	0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x55, 0x70, 0x64, 0x61, 0x74, 0x65, 0x12, 0x16, 0x0a, 0x06,
	0x75, 0x70, 0x64, 0x61, 0x74, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x75, 0x70,
	0x64, 0x61, 0x74, 0x65, 0x42, 0x35, 0x5a, 0x33, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63,
	0x6f, 0x6d, 0x2f, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x64, 0x2f, 0x63, 0x6f,
	0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x64, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x74, 0x79, 0x70,
	0x65, 0x73, 0x2f, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x66, 0x65, 0x72, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDescOnce sync.Once
	file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDescData = file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDesc
)

func file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDescGZIP() []byte {
	file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDescOnce.Do(func() {
		file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDescData = protoimpl.X.CompressGZIP(file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDescData)
	})
	return file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDescData
}

var file_github_com_containerd_containerd_api_types_transfer_streaming_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_github_com_containerd_containerd_api_types_transfer_streaming_proto_goTypes = []interface{}{
	(*Data)(nil),         // 0: containerd.types.transfer.Data
	(*WindowUpdate)(nil), // 1: containerd.types.transfer.WindowUpdate
}
var file_github_com_containerd_containerd_api_types_transfer_streaming_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_github_com_containerd_containerd_api_types_transfer_streaming_proto_init() }
func file_github_com_containerd_containerd_api_types_transfer_streaming_proto_init() {
	if File_github_com_containerd_containerd_api_types_transfer_streaming_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_github_com_containerd_containerd_api_types_transfer_streaming_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Data); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_github_com_containerd_containerd_api_types_transfer_streaming_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WindowUpdate); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_github_com_containerd_containerd_api_types_transfer_streaming_proto_goTypes,
		DependencyIndexes: file_github_com_containerd_containerd_api_types_transfer_streaming_proto_depIdxs,
		MessageInfos:      file_github_com_containerd_containerd_api_types_transfer_streaming_proto_msgTypes,
	}.Build()
	File_github_com_containerd_containerd_api_types_transfer_streaming_proto = out.File
	file_github_com_containerd_containerd_api_types_transfer_streaming_proto_rawDesc = nil
	file_github_com_containerd_containerd_api_types_transfer_streaming_proto_goTypes = nil
	file_github_com_containerd_containerd_api_types_transfer_streaming_proto_depIdxs = nil
}