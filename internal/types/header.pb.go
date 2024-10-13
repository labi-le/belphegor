// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.35.1
// 	protoc        v4.25.4
// source: header.proto

package types

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Mime int32

const (
	Mime_TEXT  Mime = 0
	Mime_IMAGE Mime = 1
)

// Enum value maps for Mime.
var (
	Mime_name = map[int32]string{
		0: "TEXT",
		1: "IMAGE",
	}
	Mime_value = map[string]int32{
		"TEXT":  0,
		"IMAGE": 1,
	}
)

func (x Mime) Enum() *Mime {
	p := new(Mime)
	*p = x
	return p
}

func (x Mime) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Mime) Descriptor() protoreflect.EnumDescriptor {
	return file_header_proto_enumTypes[0].Descriptor()
}

func (Mime) Type() protoreflect.EnumType {
	return &file_header_proto_enumTypes[0]
}

func (x Mime) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Mime.Descriptor instead.
func (Mime) EnumDescriptor() ([]byte, []int) {
	return file_header_proto_rawDescGZIP(), []int{0}
}

// ClipboardProvider represents the clipboard provider
// Used to identify the clipboard provider
type Clipboard int32

const (
	Clipboard_XClip       Clipboard = 0
	Clipboard_XSel        Clipboard = 1
	Clipboard_WlClipboard Clipboard = 2
	Clipboard_MasOsStd    Clipboard = 4
	Clipboard_WindowsNT10 Clipboard = 5
	Clipboard_Null        Clipboard = 6
)

// Enum value maps for Clipboard.
var (
	Clipboard_name = map[int32]string{
		0: "XClip",
		1: "XSel",
		2: "WlClipboard",
		4: "MasOsStd",
		5: "WindowsNT10",
		6: "Null",
	}
	Clipboard_value = map[string]int32{
		"XClip":       0,
		"XSel":        1,
		"WlClipboard": 2,
		"MasOsStd":    4,
		"WindowsNT10": 5,
		"Null":        6,
	}
)

func (x Clipboard) Enum() *Clipboard {
	p := new(Clipboard)
	*p = x
	return p
}

func (x Clipboard) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Clipboard) Descriptor() protoreflect.EnumDescriptor {
	return file_header_proto_enumTypes[1].Descriptor()
}

func (Clipboard) Type() protoreflect.EnumType {
	return &file_header_proto_enumTypes[1]
}

func (x Clipboard) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Clipboard.Descriptor instead.
func (Clipboard) EnumDescriptor() ([]byte, []int) {
	return file_header_proto_rawDescGZIP(), []int{1}
}

// Header represents the header of a message
type Header struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	From              string                 `protobuf:"bytes,1,opt,name=From,proto3" json:"From,omitempty"`
	MimeType          Mime                   `protobuf:"varint,2,opt,name=MimeType,proto3,enum=belphegor.Mime" json:"MimeType,omitempty"`
	ID                string                 `protobuf:"bytes,3,opt,name=ID,proto3" json:"ID,omitempty"`
	Created           *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=Created,proto3" json:"Created,omitempty"`
	ClipboardProvider Clipboard              `protobuf:"varint,5,opt,name=ClipboardProvider,proto3,enum=belphegor.Clipboard" json:"ClipboardProvider,omitempty"`
}

func (x *Header) Reset() {
	*x = Header{}
	mi := &file_header_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Header) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Header) ProtoMessage() {}

func (x *Header) ProtoReflect() protoreflect.Message {
	mi := &file_header_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Header.ProtoReflect.Descriptor instead.
func (*Header) Descriptor() ([]byte, []int) {
	return file_header_proto_rawDescGZIP(), []int{0}
}

func (x *Header) GetFrom() string {
	if x != nil {
		return x.From
	}
	return ""
}

func (x *Header) GetMimeType() Mime {
	if x != nil {
		return x.MimeType
	}
	return Mime_TEXT
}

func (x *Header) GetID() string {
	if x != nil {
		return x.ID
	}
	return ""
}

func (x *Header) GetCreated() *timestamppb.Timestamp {
	if x != nil {
		return x.Created
	}
	return nil
}

func (x *Header) GetClipboardProvider() Clipboard {
	if x != nil {
		return x.ClipboardProvider
	}
	return Clipboard_XClip
}

var File_header_proto protoreflect.FileDescriptor

var file_header_proto_rawDesc = []byte{
	0x0a, 0x0c, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09,
	0x62, 0x65, 0x6c, 0x70, 0x68, 0x65, 0x67, 0x6f, 0x72, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xd3, 0x01, 0x0a, 0x06, 0x48,
	0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x46, 0x72, 0x6f, 0x6d, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x46, 0x72, 0x6f, 0x6d, 0x12, 0x2b, 0x0a, 0x08, 0x4d, 0x69, 0x6d,
	0x65, 0x54, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0f, 0x2e, 0x62, 0x65,
	0x6c, 0x70, 0x68, 0x65, 0x67, 0x6f, 0x72, 0x2e, 0x4d, 0x69, 0x6d, 0x65, 0x52, 0x08, 0x4d, 0x69,
	0x6d, 0x65, 0x54, 0x79, 0x70, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x49, 0x44, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x02, 0x49, 0x44, 0x12, 0x34, 0x0a, 0x07, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65,
	0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x52, 0x07, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x12, 0x42, 0x0a, 0x11,
	0x43, 0x6c, 0x69, 0x70, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65,
	0x72, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x14, 0x2e, 0x62, 0x65, 0x6c, 0x70, 0x68, 0x65,
	0x67, 0x6f, 0x72, 0x2e, 0x43, 0x6c, 0x69, 0x70, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x52, 0x11, 0x43,
	0x6c, 0x69, 0x70, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x50, 0x72, 0x6f, 0x76, 0x69, 0x64, 0x65, 0x72,
	0x2a, 0x1b, 0x0a, 0x04, 0x4d, 0x69, 0x6d, 0x65, 0x12, 0x08, 0x0a, 0x04, 0x54, 0x45, 0x58, 0x54,
	0x10, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x49, 0x4d, 0x41, 0x47, 0x45, 0x10, 0x01, 0x2a, 0x5a, 0x0a,
	0x09, 0x43, 0x6c, 0x69, 0x70, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x12, 0x09, 0x0a, 0x05, 0x58, 0x43,
	0x6c, 0x69, 0x70, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x58, 0x53, 0x65, 0x6c, 0x10, 0x01, 0x12,
	0x0f, 0x0a, 0x0b, 0x57, 0x6c, 0x43, 0x6c, 0x69, 0x70, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x10, 0x02,
	0x12, 0x0c, 0x0a, 0x08, 0x4d, 0x61, 0x73, 0x4f, 0x73, 0x53, 0x74, 0x64, 0x10, 0x04, 0x12, 0x0f,
	0x0a, 0x0b, 0x57, 0x69, 0x6e, 0x64, 0x6f, 0x77, 0x73, 0x4e, 0x54, 0x31, 0x30, 0x10, 0x05, 0x12,
	0x08, 0x0a, 0x04, 0x4e, 0x75, 0x6c, 0x6c, 0x10, 0x06, 0x42, 0x10, 0x5a, 0x0e, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x79, 0x70, 0x65, 0x73, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_header_proto_rawDescOnce sync.Once
	file_header_proto_rawDescData = file_header_proto_rawDesc
)

func file_header_proto_rawDescGZIP() []byte {
	file_header_proto_rawDescOnce.Do(func() {
		file_header_proto_rawDescData = protoimpl.X.CompressGZIP(file_header_proto_rawDescData)
	})
	return file_header_proto_rawDescData
}

var file_header_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_header_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_header_proto_goTypes = []any{
	(Mime)(0),                     // 0: belphegor.Mime
	(Clipboard)(0),                // 1: belphegor.Clipboard
	(*Header)(nil),                // 2: belphegor.Header
	(*timestamppb.Timestamp)(nil), // 3: google.protobuf.Timestamp
}
var file_header_proto_depIdxs = []int32{
	0, // 0: belphegor.Header.MimeType:type_name -> belphegor.Mime
	3, // 1: belphegor.Header.Created:type_name -> google.protobuf.Timestamp
	1, // 2: belphegor.Header.ClipboardProvider:type_name -> belphegor.Clipboard
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_header_proto_init() }
func file_header_proto_init() {
	if File_header_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_header_proto_rawDesc,
			NumEnums:      2,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_header_proto_goTypes,
		DependencyIndexes: file_header_proto_depIdxs,
		EnumInfos:         file_header_proto_enumTypes,
		MessageInfos:      file_header_proto_msgTypes,
	}.Build()
	File_header_proto = out.File
	file_header_proto_rawDesc = nil
	file_header_proto_goTypes = nil
	file_header_proto_depIdxs = nil
}
