// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: fs.proto

package fs

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type ChecksumType int32

const (
	ChecksumType_CHECKSUM_NULL   ChecksumType = 0
	ChecksumType_CHECKSUM_CRC32  ChecksumType = 1
	ChecksumType_CHECKSUM_CRC32C ChecksumType = 2
)

var ChecksumType_name = map[int32]string{
	0: "CHECKSUM_NULL",
	1: "CHECKSUM_CRC32",
	2: "CHECKSUM_CRC32C",
}

var ChecksumType_value = map[string]int32{
	"CHECKSUM_NULL":   0,
	"CHECKSUM_CRC32":  1,
	"CHECKSUM_CRC32C": 2,
}

func (x ChecksumType) String() string {
	return proto.EnumName(ChecksumType_name, int32(x))
}

func (ChecksumType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_e604833c2b457e38, []int{0}
}

type Page struct {
	Uid                  string            `protobuf:"bytes,1,opt,name=uid,proto3" json:"uid,omitempty"`
	UsedSpace            uint64            `protobuf:"varint,2,opt,name=usedSpace,proto3" json:"usedSpace,omitempty"`
	Blocks               map[string]*Block `protobuf:"bytes,3,rep,name=blocks,proto3" json:"blocks,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	ChecksumType         ChecksumType      `protobuf:"varint,4,opt,name=checksum_type,json=checksumType,proto3,enum=fs_pb.ChecksumType" json:"checksum_type,omitempty"`
	Checksum             string            `protobuf:"bytes,5,opt,name=checksum,proto3" json:"checksum,omitempty"`
	UpdatedAt            int64             `protobuf:"varint,6,opt,name=updated_at,json=updatedAt,proto3" json:"updated_at,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *Page) Reset()         { *m = Page{} }
func (m *Page) String() string { return proto.CompactTextString(m) }
func (*Page) ProtoMessage()    {}
func (*Page) Descriptor() ([]byte, []int) {
	return fileDescriptor_e604833c2b457e38, []int{0}
}
func (m *Page) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Page) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Page.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Page) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Page.Merge(m, src)
}
func (m *Page) XXX_Size() int {
	return m.Size()
}
func (m *Page) XXX_DiscardUnknown() {
	xxx_messageInfo_Page.DiscardUnknown(m)
}

var xxx_messageInfo_Page proto.InternalMessageInfo

func (m *Page) GetUid() string {
	if m != nil {
		return m.Uid
	}
	return ""
}

func (m *Page) GetUsedSpace() uint64 {
	if m != nil {
		return m.UsedSpace
	}
	return 0
}

func (m *Page) GetBlocks() map[string]*Block {
	if m != nil {
		return m.Blocks
	}
	return nil
}

func (m *Page) GetChecksumType() ChecksumType {
	if m != nil {
		return m.ChecksumType
	}
	return ChecksumType_CHECKSUM_NULL
}

func (m *Page) GetChecksum() string {
	if m != nil {
		return m.Checksum
	}
	return ""
}

func (m *Page) GetUpdatedAt() int64 {
	if m != nil {
		return m.UpdatedAt
	}
	return 0
}

type File struct {
	Uid                  string   `protobuf:"bytes,1,opt,name=uid,proto3" json:"uid,omitempty"`
	Name                 string   `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	DirID                string   `protobuf:"bytes,3,opt,name=dirID,proto3" json:"dirID,omitempty"`
	FullPath             string   `protobuf:"bytes,4,opt,name=fullPath,proto3" json:"fullPath,omitempty"`
	Timestamp            int64    `protobuf:"varint,5,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Size_                uint64   `protobuf:"varint,6,opt,name=size,proto3" json:"size,omitempty"`
	Blocks               int32    `protobuf:"varint,7,opt,name=blocks,proto3" json:"blocks,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *File) Reset()         { *m = File{} }
func (m *File) String() string { return proto.CompactTextString(m) }
func (*File) ProtoMessage()    {}
func (*File) Descriptor() ([]byte, []int) {
	return fileDescriptor_e604833c2b457e38, []int{1}
}
func (m *File) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *File) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_File.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *File) XXX_Merge(src proto.Message) {
	xxx_messageInfo_File.Merge(m, src)
}
func (m *File) XXX_Size() int {
	return m.Size()
}
func (m *File) XXX_DiscardUnknown() {
	xxx_messageInfo_File.DiscardUnknown(m)
}

var xxx_messageInfo_File proto.InternalMessageInfo

func (m *File) GetUid() string {
	if m != nil {
		return m.Uid
	}
	return ""
}

func (m *File) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *File) GetDirID() string {
	if m != nil {
		return m.DirID
	}
	return ""
}

func (m *File) GetFullPath() string {
	if m != nil {
		return m.FullPath
	}
	return ""
}

func (m *File) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *File) GetSize_() uint64 {
	if m != nil {
		return m.Size_
	}
	return 0
}

func (m *File) GetBlocks() int32 {
	if m != nil {
		return m.Blocks
	}
	return 0
}

type Block struct {
	Uid                  string   `protobuf:"bytes,1,opt,name=uid,proto3" json:"uid,omitempty"`
	FileId               string   `protobuf:"bytes,2,opt,name=file_id,json=fileId,proto3" json:"file_id,omitempty"`
	Size_                uint64   `protobuf:"varint,3,opt,name=size,proto3" json:"size,omitempty"`
	Data                 []byte   `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Block) Reset()         { *m = Block{} }
func (m *Block) String() string { return proto.CompactTextString(m) }
func (*Block) ProtoMessage()    {}
func (*Block) Descriptor() ([]byte, []int) {
	return fileDescriptor_e604833c2b457e38, []int{2}
}
func (m *Block) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Block) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Block.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Block) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Block.Merge(m, src)
}
func (m *Block) XXX_Size() int {
	return m.Size()
}
func (m *Block) XXX_DiscardUnknown() {
	xxx_messageInfo_Block.DiscardUnknown(m)
}

var xxx_messageInfo_Block proto.InternalMessageInfo

func (m *Block) GetUid() string {
	if m != nil {
		return m.Uid
	}
	return ""
}

func (m *Block) GetFileId() string {
	if m != nil {
		return m.FileId
	}
	return ""
}

func (m *Block) GetSize_() uint64 {
	if m != nil {
		return m.Size_
	}
	return 0
}

func (m *Block) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func init() {
	proto.RegisterEnum("fs_pb.ChecksumType", ChecksumType_name, ChecksumType_value)
	proto.RegisterType((*Page)(nil), "fs_pb.Page")
	proto.RegisterMapType((map[string]*Block)(nil), "fs_pb.Page.BlocksEntry")
	proto.RegisterType((*File)(nil), "fs_pb.File")
	proto.RegisterType((*Block)(nil), "fs_pb.Block")
}

func init() { proto.RegisterFile("fs.proto", fileDescriptor_e604833c2b457e38) }

var fileDescriptor_e604833c2b457e38 = []byte{
	// 464 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x52, 0xc1, 0x6e, 0xd3, 0x40,
	0x14, 0xec, 0xc6, 0x76, 0x5a, 0xbf, 0xa4, 0x25, 0x6c, 0x11, 0xb5, 0x2a, 0x88, 0xac, 0x9c, 0x0c,
	0x07, 0x47, 0x4a, 0x2f, 0x15, 0x9c, 0xa8, 0x29, 0x50, 0x28, 0xa8, 0xda, 0xd2, 0x4b, 0x2f, 0xd6,
	0xda, 0x5e, 0x37, 0x56, 0xec, 0x7a, 0xe5, 0xdd, 0x45, 0x32, 0x5f, 0xc2, 0x0f, 0x20, 0xf1, 0x29,
	0x1c, 0xf9, 0x04, 0x14, 0x7e, 0x04, 0x79, 0xe3, 0xb8, 0xa9, 0xd4, 0x93, 0x67, 0xe6, 0x79, 0xc7,
	0x33, 0x6f, 0x0d, 0x3b, 0xa9, 0xf0, 0x79, 0x55, 0xca, 0x12, 0x5b, 0xa9, 0x08, 0x79, 0x34, 0xf9,
	0xd9, 0x03, 0xf3, 0x82, 0xde, 0x30, 0x3c, 0x02, 0x43, 0x65, 0x89, 0x83, 0x5c, 0xe4, 0xd9, 0xa4,
	0x81, 0xf8, 0x19, 0xd8, 0x4a, 0xb0, 0xe4, 0x92, 0xd3, 0x98, 0x39, 0x3d, 0x17, 0x79, 0x26, 0xb9,
	0x13, 0xf0, 0x14, 0xfa, 0x51, 0x5e, 0xc6, 0x0b, 0xe1, 0x18, 0xae, 0xe1, 0x0d, 0x66, 0x07, 0xbe,
	0x36, 0xf4, 0x1b, 0x33, 0xff, 0x44, 0x4f, 0x4e, 0x6f, 0x65, 0x55, 0x93, 0xf6, 0x35, 0x7c, 0x0c,
	0xbb, 0xf1, 0x9c, 0xc5, 0x0b, 0xa1, 0x8a, 0x50, 0xd6, 0x9c, 0x39, 0xa6, 0x8b, 0xbc, 0xbd, 0xd9,
	0x7e, 0x7b, 0x2e, 0x68, 0x67, 0x5f, 0x6b, 0xce, 0xc8, 0x30, 0xde, 0x60, 0xf8, 0x10, 0x76, 0xd6,
	0xdc, 0xb1, 0x74, 0xbe, 0x8e, 0xe3, 0xe7, 0x00, 0x8a, 0x27, 0x54, 0xb2, 0x24, 0xa4, 0xd2, 0xe9,
	0xbb, 0xc8, 0x33, 0x88, 0xdd, 0x2a, 0x6f, 0xe4, 0xe1, 0x7b, 0x18, 0x6c, 0x64, 0x69, 0x4a, 0x2e,
	0x58, 0xbd, 0x2e, 0xb9, 0x60, 0x35, 0x9e, 0x80, 0xf5, 0x8d, 0xe6, 0x6a, 0x55, 0x70, 0x30, 0x1b,
	0xb6, 0x69, 0xf4, 0x21, 0xb2, 0x1a, 0xbd, 0xea, 0x1d, 0xa3, 0xc9, 0x2f, 0x04, 0xe6, 0xbb, 0x2c,
	0x7f, 0x68, 0x4f, 0x18, 0xcc, 0x5b, 0x5a, 0xac, 0x1c, 0x6c, 0xa2, 0x31, 0x7e, 0x02, 0x56, 0x92,
	0x55, 0x67, 0x6f, 0x1d, 0x43, 0x8b, 0x2b, 0xd2, 0x14, 0x49, 0x55, 0x9e, 0x5f, 0x50, 0x39, 0xd7,
	0xed, 0x6d, 0xd2, 0xf1, 0x66, 0xdb, 0x32, 0x2b, 0x98, 0x90, 0xb4, 0xe0, 0xba, 0xa5, 0x41, 0xee,
	0x84, 0xe6, 0x1b, 0x22, 0xfb, 0xce, 0x74, 0x41, 0x93, 0x68, 0x8c, 0x9f, 0x76, 0x37, 0xb0, 0xed,
	0x22, 0xcf, 0x5a, 0x2f, 0x7a, 0x72, 0x0d, 0x96, 0x8e, 0xff, 0x40, 0xd4, 0x03, 0xd8, 0x4e, 0xb3,
	0x9c, 0x85, 0x59, 0xd2, 0xa6, 0xed, 0x37, 0xf4, 0x2c, 0xe9, 0xfc, 0x8d, 0x0d, 0x7f, 0x0c, 0x66,
	0x42, 0x25, 0xd5, 0x49, 0x87, 0x44, 0xe3, 0x97, 0x1f, 0x61, 0xb8, 0x79, 0x51, 0xf8, 0x31, 0xec,
	0x06, 0x1f, 0x4e, 0x83, 0x4f, 0x97, 0x57, 0x9f, 0xc3, 0x2f, 0x57, 0xe7, 0xe7, 0xa3, 0x2d, 0x8c,
	0x61, 0xaf, 0x93, 0x02, 0x12, 0x1c, 0xcd, 0x46, 0x08, 0xef, 0xc3, 0xa3, 0xfb, 0x5a, 0x30, 0xea,
	0x9d, 0xbc, 0xfe, 0xbd, 0x1c, 0xa3, 0x3f, 0xcb, 0x31, 0xfa, 0xbb, 0x1c, 0xa3, 0x1f, 0xff, 0xc6,
	0x5b, 0xd7, 0x2f, 0x6e, 0x32, 0x39, 0x57, 0x91, 0x1f, 0x97, 0xc5, 0x94, 0xd3, 0x8a, 0x8a, 0x52,
	0x55, 0x31, 0x9b, 0x56, 0xf3, 0x52, 0x28, 0xb1, 0x7e, 0xf0, 0x68, 0x9a, 0x8a, 0xa8, 0xaf, 0xff,
	0xe2, 0xa3, 0xff, 0x01, 0x00, 0x00, 0xff, 0xff, 0x1c, 0xed, 0xb5, 0x5c, 0xd1, 0x02, 0x00, 0x00,
}

func (m *Page) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Page) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Page) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.UpdatedAt != 0 {
		i = encodeVarintFs(dAtA, i, uint64(m.UpdatedAt))
		i--
		dAtA[i] = 0x30
	}
	if len(m.Checksum) > 0 {
		i -= len(m.Checksum)
		copy(dAtA[i:], m.Checksum)
		i = encodeVarintFs(dAtA, i, uint64(len(m.Checksum)))
		i--
		dAtA[i] = 0x2a
	}
	if m.ChecksumType != 0 {
		i = encodeVarintFs(dAtA, i, uint64(m.ChecksumType))
		i--
		dAtA[i] = 0x20
	}
	if len(m.Blocks) > 0 {
		for k := range m.Blocks {
			v := m.Blocks[k]
			baseI := i
			if v != nil {
				{
					size, err := v.MarshalToSizedBuffer(dAtA[:i])
					if err != nil {
						return 0, err
					}
					i -= size
					i = encodeVarintFs(dAtA, i, uint64(size))
				}
				i--
				dAtA[i] = 0x12
			}
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintFs(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintFs(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x1a
		}
	}
	if m.UsedSpace != 0 {
		i = encodeVarintFs(dAtA, i, uint64(m.UsedSpace))
		i--
		dAtA[i] = 0x10
	}
	if len(m.Uid) > 0 {
		i -= len(m.Uid)
		copy(dAtA[i:], m.Uid)
		i = encodeVarintFs(dAtA, i, uint64(len(m.Uid)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *File) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *File) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *File) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Blocks != 0 {
		i = encodeVarintFs(dAtA, i, uint64(m.Blocks))
		i--
		dAtA[i] = 0x38
	}
	if m.Size_ != 0 {
		i = encodeVarintFs(dAtA, i, uint64(m.Size_))
		i--
		dAtA[i] = 0x30
	}
	if m.Timestamp != 0 {
		i = encodeVarintFs(dAtA, i, uint64(m.Timestamp))
		i--
		dAtA[i] = 0x28
	}
	if len(m.FullPath) > 0 {
		i -= len(m.FullPath)
		copy(dAtA[i:], m.FullPath)
		i = encodeVarintFs(dAtA, i, uint64(len(m.FullPath)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.DirID) > 0 {
		i -= len(m.DirID)
		copy(dAtA[i:], m.DirID)
		i = encodeVarintFs(dAtA, i, uint64(len(m.DirID)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintFs(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Uid) > 0 {
		i -= len(m.Uid)
		copy(dAtA[i:], m.Uid)
		i = encodeVarintFs(dAtA, i, uint64(len(m.Uid)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *Block) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Block) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Block) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Data) > 0 {
		i -= len(m.Data)
		copy(dAtA[i:], m.Data)
		i = encodeVarintFs(dAtA, i, uint64(len(m.Data)))
		i--
		dAtA[i] = 0x22
	}
	if m.Size_ != 0 {
		i = encodeVarintFs(dAtA, i, uint64(m.Size_))
		i--
		dAtA[i] = 0x18
	}
	if len(m.FileId) > 0 {
		i -= len(m.FileId)
		copy(dAtA[i:], m.FileId)
		i = encodeVarintFs(dAtA, i, uint64(len(m.FileId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Uid) > 0 {
		i -= len(m.Uid)
		copy(dAtA[i:], m.Uid)
		i = encodeVarintFs(dAtA, i, uint64(len(m.Uid)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintFs(dAtA []byte, offset int, v uint64) int {
	offset -= sovFs(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Page) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Uid)
	if l > 0 {
		n += 1 + l + sovFs(uint64(l))
	}
	if m.UsedSpace != 0 {
		n += 1 + sovFs(uint64(m.UsedSpace))
	}
	if len(m.Blocks) > 0 {
		for k, v := range m.Blocks {
			_ = k
			_ = v
			l = 0
			if v != nil {
				l = v.Size()
				l += 1 + sovFs(uint64(l))
			}
			mapEntrySize := 1 + len(k) + sovFs(uint64(len(k))) + l
			n += mapEntrySize + 1 + sovFs(uint64(mapEntrySize))
		}
	}
	if m.ChecksumType != 0 {
		n += 1 + sovFs(uint64(m.ChecksumType))
	}
	l = len(m.Checksum)
	if l > 0 {
		n += 1 + l + sovFs(uint64(l))
	}
	if m.UpdatedAt != 0 {
		n += 1 + sovFs(uint64(m.UpdatedAt))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *File) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Uid)
	if l > 0 {
		n += 1 + l + sovFs(uint64(l))
	}
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovFs(uint64(l))
	}
	l = len(m.DirID)
	if l > 0 {
		n += 1 + l + sovFs(uint64(l))
	}
	l = len(m.FullPath)
	if l > 0 {
		n += 1 + l + sovFs(uint64(l))
	}
	if m.Timestamp != 0 {
		n += 1 + sovFs(uint64(m.Timestamp))
	}
	if m.Size_ != 0 {
		n += 1 + sovFs(uint64(m.Size_))
	}
	if m.Blocks != 0 {
		n += 1 + sovFs(uint64(m.Blocks))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *Block) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Uid)
	if l > 0 {
		n += 1 + l + sovFs(uint64(l))
	}
	l = len(m.FileId)
	if l > 0 {
		n += 1 + l + sovFs(uint64(l))
	}
	if m.Size_ != 0 {
		n += 1 + sovFs(uint64(m.Size_))
	}
	l = len(m.Data)
	if l > 0 {
		n += 1 + l + sovFs(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovFs(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozFs(x uint64) (n int) {
	return sovFs(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Page) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFs
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Page: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Page: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Uid", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthFs
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFs
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Uid = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field UsedSpace", wireType)
			}
			m.UsedSpace = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.UsedSpace |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Blocks", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthFs
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthFs
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Blocks == nil {
				m.Blocks = make(map[string]*Block)
			}
			var mapkey string
			var mapvalue *Block
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowFs
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowFs
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthFs
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthFs
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var mapmsglen int
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowFs
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						mapmsglen |= int(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					if mapmsglen < 0 {
						return ErrInvalidLengthFs
					}
					postmsgIndex := iNdEx + mapmsglen
					if postmsgIndex < 0 {
						return ErrInvalidLengthFs
					}
					if postmsgIndex > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = &Block{}
					if err := mapvalue.Unmarshal(dAtA[iNdEx:postmsgIndex]); err != nil {
						return err
					}
					iNdEx = postmsgIndex
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipFs(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthFs
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Blocks[mapkey] = mapvalue
			iNdEx = postIndex
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ChecksumType", wireType)
			}
			m.ChecksumType = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ChecksumType |= ChecksumType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Checksum", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthFs
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFs
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Checksum = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field UpdatedAt", wireType)
			}
			m.UpdatedAt = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.UpdatedAt |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipFs(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthFs
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *File) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFs
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: File: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: File: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Uid", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthFs
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFs
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Uid = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthFs
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFs
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field DirID", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthFs
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFs
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.DirID = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FullPath", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthFs
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFs
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FullPath = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Timestamp", wireType)
			}
			m.Timestamp = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Timestamp |= int64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Size_", wireType)
			}
			m.Size_ = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Size_ |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Blocks", wireType)
			}
			m.Blocks = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Blocks |= int32(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipFs(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthFs
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *Block) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowFs
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Block: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Block: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Uid", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthFs
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFs
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Uid = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field FileId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthFs
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthFs
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.FileId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Size_", wireType)
			}
			m.Size_ = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Size_ |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Data", wireType)
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowFs
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				byteLen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if byteLen < 0 {
				return ErrInvalidLengthFs
			}
			postIndex := iNdEx + byteLen
			if postIndex < 0 {
				return ErrInvalidLengthFs
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Data = append(m.Data[:0], dAtA[iNdEx:postIndex]...)
			if m.Data == nil {
				m.Data = []byte{}
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipFs(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthFs
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipFs(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowFs
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowFs
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowFs
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthFs
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupFs
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthFs
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthFs        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowFs          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupFs = fmt.Errorf("proto: unexpected end of group")
)
