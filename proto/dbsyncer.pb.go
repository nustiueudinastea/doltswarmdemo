// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        (unknown)
// source: proto/dbsyncer.proto

package proto

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

type AdvertiseHeadRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Head string `protobuf:"bytes,1,opt,name=head,proto3" json:"head,omitempty"`
}

func (x *AdvertiseHeadRequest) Reset() {
	*x = AdvertiseHeadRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_dbsyncer_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AdvertiseHeadRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AdvertiseHeadRequest) ProtoMessage() {}

func (x *AdvertiseHeadRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_dbsyncer_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AdvertiseHeadRequest.ProtoReflect.Descriptor instead.
func (*AdvertiseHeadRequest) Descriptor() ([]byte, []int) {
	return file_proto_dbsyncer_proto_rawDescGZIP(), []int{0}
}

func (x *AdvertiseHeadRequest) GetHead() string {
	if x != nil {
		return x.Head
	}
	return ""
}

type AdvertiseHeadResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *AdvertiseHeadResponse) Reset() {
	*x = AdvertiseHeadResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_dbsyncer_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AdvertiseHeadResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AdvertiseHeadResponse) ProtoMessage() {}

func (x *AdvertiseHeadResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_dbsyncer_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AdvertiseHeadResponse.ProtoReflect.Descriptor instead.
func (*AdvertiseHeadResponse) Descriptor() ([]byte, []int) {
	return file_proto_dbsyncer_proto_rawDescGZIP(), []int{1}
}

var File_proto_dbsyncer_proto protoreflect.FileDescriptor

var file_proto_dbsyncer_proto_rawDesc = []byte{
	0x0a, 0x14, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x64, 0x62, 0x73, 0x79, 0x6e, 0x63, 0x65, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x2a, 0x0a,
	0x14, 0x41, 0x64, 0x76, 0x65, 0x72, 0x74, 0x69, 0x73, 0x65, 0x48, 0x65, 0x61, 0x64, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x65, 0x61, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x68, 0x65, 0x61, 0x64, 0x22, 0x17, 0x0a, 0x15, 0x41, 0x64, 0x76,
	0x65, 0x72, 0x74, 0x69, 0x73, 0x65, 0x48, 0x65, 0x61, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x32, 0x58, 0x0a, 0x08, 0x44, 0x42, 0x53, 0x79, 0x6e, 0x63, 0x65, 0x72, 0x12, 0x4c,
	0x0a, 0x0d, 0x41, 0x64, 0x76, 0x65, 0x72, 0x74, 0x69, 0x73, 0x65, 0x48, 0x65, 0x61, 0x64, 0x12,
	0x1b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x41, 0x64, 0x76, 0x65, 0x72, 0x74, 0x69, 0x73,
	0x65, 0x48, 0x65, 0x61, 0x64, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1c, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x41, 0x64, 0x76, 0x65, 0x72, 0x74, 0x69, 0x73, 0x65, 0x48, 0x65,
	0x61, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x09, 0x5a, 0x07,
	0x2e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_dbsyncer_proto_rawDescOnce sync.Once
	file_proto_dbsyncer_proto_rawDescData = file_proto_dbsyncer_proto_rawDesc
)

func file_proto_dbsyncer_proto_rawDescGZIP() []byte {
	file_proto_dbsyncer_proto_rawDescOnce.Do(func() {
		file_proto_dbsyncer_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_dbsyncer_proto_rawDescData)
	})
	return file_proto_dbsyncer_proto_rawDescData
}

var file_proto_dbsyncer_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_proto_dbsyncer_proto_goTypes = []interface{}{
	(*AdvertiseHeadRequest)(nil),  // 0: proto.AdvertiseHeadRequest
	(*AdvertiseHeadResponse)(nil), // 1: proto.AdvertiseHeadResponse
}
var file_proto_dbsyncer_proto_depIdxs = []int32{
	0, // 0: proto.DBSyncer.AdvertiseHead:input_type -> proto.AdvertiseHeadRequest
	1, // 1: proto.DBSyncer.AdvertiseHead:output_type -> proto.AdvertiseHeadResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_proto_dbsyncer_proto_init() }
func file_proto_dbsyncer_proto_init() {
	if File_proto_dbsyncer_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_dbsyncer_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AdvertiseHeadRequest); i {
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
		file_proto_dbsyncer_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AdvertiseHeadResponse); i {
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
			RawDescriptor: file_proto_dbsyncer_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_dbsyncer_proto_goTypes,
		DependencyIndexes: file_proto_dbsyncer_proto_depIdxs,
		MessageInfos:      file_proto_dbsyncer_proto_msgTypes,
	}.Build()
	File_proto_dbsyncer_proto = out.File
	file_proto_dbsyncer_proto_rawDesc = nil
	file_proto_dbsyncer_proto_goTypes = nil
	file_proto_dbsyncer_proto_depIdxs = nil
}
