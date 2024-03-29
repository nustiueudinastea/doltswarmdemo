// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.31.0
// 	protoc        (unknown)
// source: p2p/proto/tester.proto

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

type ExecSQLRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Statement string `protobuf:"bytes,1,opt,name=statement,proto3" json:"statement,omitempty"`
	Msg       string `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
}

func (x *ExecSQLRequest) Reset() {
	*x = ExecSQLRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_p2p_proto_tester_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExecSQLRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExecSQLRequest) ProtoMessage() {}

func (x *ExecSQLRequest) ProtoReflect() protoreflect.Message {
	mi := &file_p2p_proto_tester_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExecSQLRequest.ProtoReflect.Descriptor instead.
func (*ExecSQLRequest) Descriptor() ([]byte, []int) {
	return file_p2p_proto_tester_proto_rawDescGZIP(), []int{0}
}

func (x *ExecSQLRequest) GetStatement() string {
	if x != nil {
		return x.Statement
	}
	return ""
}

func (x *ExecSQLRequest) GetMsg() string {
	if x != nil {
		return x.Msg
	}
	return ""
}

type ExecSQLResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Commit string `protobuf:"bytes,1,opt,name=commit,proto3" json:"commit,omitempty"`
	Result string `protobuf:"bytes,2,opt,name=result,proto3" json:"result,omitempty"`
	Err    string `protobuf:"bytes,3,opt,name=err,proto3" json:"err,omitempty"`
}

func (x *ExecSQLResponse) Reset() {
	*x = ExecSQLResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_p2p_proto_tester_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ExecSQLResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ExecSQLResponse) ProtoMessage() {}

func (x *ExecSQLResponse) ProtoReflect() protoreflect.Message {
	mi := &file_p2p_proto_tester_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ExecSQLResponse.ProtoReflect.Descriptor instead.
func (*ExecSQLResponse) Descriptor() ([]byte, []int) {
	return file_p2p_proto_tester_proto_rawDescGZIP(), []int{1}
}

func (x *ExecSQLResponse) GetCommit() string {
	if x != nil {
		return x.Commit
	}
	return ""
}

func (x *ExecSQLResponse) GetResult() string {
	if x != nil {
		return x.Result
	}
	return ""
}

func (x *ExecSQLResponse) GetErr() string {
	if x != nil {
		return x.Err
	}
	return ""
}

type GetAllCommitsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetAllCommitsRequest) Reset() {
	*x = GetAllCommitsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_p2p_proto_tester_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetAllCommitsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAllCommitsRequest) ProtoMessage() {}

func (x *GetAllCommitsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_p2p_proto_tester_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAllCommitsRequest.ProtoReflect.Descriptor instead.
func (*GetAllCommitsRequest) Descriptor() ([]byte, []int) {
	return file_p2p_proto_tester_proto_rawDescGZIP(), []int{2}
}

type GetAllCommitsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Commits []string `protobuf:"bytes,1,rep,name=commits,proto3" json:"commits,omitempty"`
}

func (x *GetAllCommitsResponse) Reset() {
	*x = GetAllCommitsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_p2p_proto_tester_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetAllCommitsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetAllCommitsResponse) ProtoMessage() {}

func (x *GetAllCommitsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_p2p_proto_tester_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetAllCommitsResponse.ProtoReflect.Descriptor instead.
func (*GetAllCommitsResponse) Descriptor() ([]byte, []int) {
	return file_p2p_proto_tester_proto_rawDescGZIP(), []int{3}
}

func (x *GetAllCommitsResponse) GetCommits() []string {
	if x != nil {
		return x.Commits
	}
	return nil
}

type GetHeadRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *GetHeadRequest) Reset() {
	*x = GetHeadRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_p2p_proto_tester_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetHeadRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetHeadRequest) ProtoMessage() {}

func (x *GetHeadRequest) ProtoReflect() protoreflect.Message {
	mi := &file_p2p_proto_tester_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetHeadRequest.ProtoReflect.Descriptor instead.
func (*GetHeadRequest) Descriptor() ([]byte, []int) {
	return file_p2p_proto_tester_proto_rawDescGZIP(), []int{4}
}

type GetHeadResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Commit string `protobuf:"bytes,1,opt,name=commit,proto3" json:"commit,omitempty"`
}

func (x *GetHeadResponse) Reset() {
	*x = GetHeadResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_p2p_proto_tester_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetHeadResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetHeadResponse) ProtoMessage() {}

func (x *GetHeadResponse) ProtoReflect() protoreflect.Message {
	mi := &file_p2p_proto_tester_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetHeadResponse.ProtoReflect.Descriptor instead.
func (*GetHeadResponse) Descriptor() ([]byte, []int) {
	return file_p2p_proto_tester_proto_rawDescGZIP(), []int{5}
}

func (x *GetHeadResponse) GetCommit() string {
	if x != nil {
		return x.Commit
	}
	return ""
}

var File_p2p_proto_tester_proto protoreflect.FileDescriptor

var file_p2p_proto_tester_proto_rawDesc = []byte{
	0x0a, 0x16, 0x70, 0x32, 0x70, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x74, 0x65, 0x73, 0x74,
	0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22,
	0x40, 0x0a, 0x0e, 0x45, 0x78, 0x65, 0x63, 0x53, 0x51, 0x4c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x1c, 0x0a, 0x09, 0x73, 0x74, 0x61, 0x74, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x73, 0x74, 0x61, 0x74, 0x65, 0x6d, 0x65, 0x6e, 0x74, 0x12,
	0x10, 0x0a, 0x03, 0x6d, 0x73, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6d, 0x73,
	0x67, 0x22, 0x53, 0x0a, 0x0f, 0x45, 0x78, 0x65, 0x63, 0x53, 0x51, 0x4c, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x12, 0x16, 0x0a, 0x06,
	0x72, 0x65, 0x73, 0x75, 0x6c, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x72, 0x65,
	0x73, 0x75, 0x6c, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x65, 0x72, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x65, 0x72, 0x72, 0x22, 0x16, 0x0a, 0x14, 0x47, 0x65, 0x74, 0x41, 0x6c, 0x6c,
	0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x22, 0x31,
	0x0a, 0x15, 0x47, 0x65, 0x74, 0x41, 0x6c, 0x6c, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x69,
	0x74, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x73, 0x22, 0x10, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x48, 0x65, 0x61, 0x64, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x22, 0x29, 0x0a, 0x0f, 0x47, 0x65, 0x74, 0x48, 0x65, 0x61, 0x64, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x32, 0xce,
	0x01, 0x0a, 0x06, 0x54, 0x65, 0x73, 0x74, 0x65, 0x72, 0x12, 0x3a, 0x0a, 0x07, 0x45, 0x78, 0x65,
	0x63, 0x53, 0x51, 0x4c, 0x12, 0x15, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x45, 0x78, 0x65,
	0x63, 0x53, 0x51, 0x4c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x2e, 0x45, 0x78, 0x65, 0x63, 0x53, 0x51, 0x4c, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x4c, 0x0a, 0x0d, 0x47, 0x65, 0x74, 0x41, 0x6c, 0x6c, 0x43,
	0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x12, 0x1b, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47,
	0x65, 0x74, 0x41, 0x6c, 0x6c, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x1c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65, 0x74, 0x41,
	0x6c, 0x6c, 0x43, 0x6f, 0x6d, 0x6d, 0x69, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x00, 0x12, 0x3a, 0x0a, 0x07, 0x47, 0x65, 0x74, 0x48, 0x65, 0x61, 0x64, 0x12, 0x15,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65, 0x74, 0x48, 0x65, 0x61, 0x64, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x47, 0x65,
	0x74, 0x48, 0x65, 0x61, 0x64, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42,
	0x09, 0x5a, 0x07, 0x2e, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_p2p_proto_tester_proto_rawDescOnce sync.Once
	file_p2p_proto_tester_proto_rawDescData = file_p2p_proto_tester_proto_rawDesc
)

func file_p2p_proto_tester_proto_rawDescGZIP() []byte {
	file_p2p_proto_tester_proto_rawDescOnce.Do(func() {
		file_p2p_proto_tester_proto_rawDescData = protoimpl.X.CompressGZIP(file_p2p_proto_tester_proto_rawDescData)
	})
	return file_p2p_proto_tester_proto_rawDescData
}

var file_p2p_proto_tester_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_p2p_proto_tester_proto_goTypes = []interface{}{
	(*ExecSQLRequest)(nil),        // 0: proto.ExecSQLRequest
	(*ExecSQLResponse)(nil),       // 1: proto.ExecSQLResponse
	(*GetAllCommitsRequest)(nil),  // 2: proto.GetAllCommitsRequest
	(*GetAllCommitsResponse)(nil), // 3: proto.GetAllCommitsResponse
	(*GetHeadRequest)(nil),        // 4: proto.GetHeadRequest
	(*GetHeadResponse)(nil),       // 5: proto.GetHeadResponse
}
var file_p2p_proto_tester_proto_depIdxs = []int32{
	0, // 0: proto.Tester.ExecSQL:input_type -> proto.ExecSQLRequest
	2, // 1: proto.Tester.GetAllCommits:input_type -> proto.GetAllCommitsRequest
	4, // 2: proto.Tester.GetHead:input_type -> proto.GetHeadRequest
	1, // 3: proto.Tester.ExecSQL:output_type -> proto.ExecSQLResponse
	3, // 4: proto.Tester.GetAllCommits:output_type -> proto.GetAllCommitsResponse
	5, // 5: proto.Tester.GetHead:output_type -> proto.GetHeadResponse
	3, // [3:6] is the sub-list for method output_type
	0, // [0:3] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_p2p_proto_tester_proto_init() }
func file_p2p_proto_tester_proto_init() {
	if File_p2p_proto_tester_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_p2p_proto_tester_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExecSQLRequest); i {
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
		file_p2p_proto_tester_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ExecSQLResponse); i {
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
		file_p2p_proto_tester_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetAllCommitsRequest); i {
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
		file_p2p_proto_tester_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetAllCommitsResponse); i {
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
		file_p2p_proto_tester_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetHeadRequest); i {
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
		file_p2p_proto_tester_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetHeadResponse); i {
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
			RawDescriptor: file_p2p_proto_tester_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_p2p_proto_tester_proto_goTypes,
		DependencyIndexes: file_p2p_proto_tester_proto_depIdxs,
		MessageInfos:      file_p2p_proto_tester_proto_msgTypes,
	}.Build()
	File_p2p_proto_tester_proto = out.File
	file_p2p_proto_tester_proto_rawDesc = nil
	file_p2p_proto_tester_proto_goTypes = nil
	file_p2p_proto_tester_proto_depIdxs = nil
}
