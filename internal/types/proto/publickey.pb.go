// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v6.31.1
// source: publickey.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type PublicKey struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Key           []byte                 `protobuf:"bytes,1,opt,name=Key,proto3" json:"Key,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *PublicKey) Reset() {
	*x = PublicKey{}
	mi := &file_publickey_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *PublicKey) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PublicKey) ProtoMessage() {}

func (x *PublicKey) ProtoReflect() protoreflect.Message {
	mi := &file_publickey_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PublicKey.ProtoReflect.Descriptor instead.
func (*PublicKey) Descriptor() ([]byte, []int) {
	return file_publickey_proto_rawDescGZIP(), []int{0}
}

func (x *PublicKey) GetKey() []byte {
	if x != nil {
		return x.Key
	}
	return nil
}

var File_publickey_proto protoreflect.FileDescriptor

const file_publickey_proto_rawDesc = "" +
	"\n" +
	"\x0fpublickey.proto\x12\tbelphegor\"\x1d\n" +
	"\tPublicKey\x12\x10\n" +
	"\x03Key\x18\x01 \x01(\fR\x03KeyB\x16Z\x14internal/types/protob\x06proto3"

var (
	file_publickey_proto_rawDescOnce sync.Once
	file_publickey_proto_rawDescData []byte
)

func file_publickey_proto_rawDescGZIP() []byte {
	file_publickey_proto_rawDescOnce.Do(func() {
		file_publickey_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_publickey_proto_rawDesc), len(file_publickey_proto_rawDesc)))
	})
	return file_publickey_proto_rawDescData
}

var file_publickey_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_publickey_proto_goTypes = []any{
	(*PublicKey)(nil), // 0: belphegor.PublicKey
}
var file_publickey_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_publickey_proto_init() }
func file_publickey_proto_init() {
	if File_publickey_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_publickey_proto_rawDesc), len(file_publickey_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_publickey_proto_goTypes,
		DependencyIndexes: file_publickey_proto_depIdxs,
		MessageInfos:      file_publickey_proto_msgTypes,
	}.Build()
	File_publickey_proto = out.File
	file_publickey_proto_goTypes = nil
	file_publickey_proto_depIdxs = nil
}
