package so

import (
	"bytes"
	"math"
	"sync"
)

const (
	FlagIsCompressed        = 0x01
	FlagHasName             = 0x02
	FlagHasMime             = 0x04
	FlagHasLastModifiedDate = 0x08
	FlagHasTtl              = 0x10
	FlagHasMeta             = 0x20
	LastModifiedBytesLength = 5
	TtlBytesLength          = 2
)

const (
	SizeSize           = 4 // uint32 size
	NeedleIdSize       = 8
	OffsetSize         = 4
	NeedleHeaderSize   = CookieSize + NeedleIdSize + SizeSize
	NeedleMapEntrySize = NeedleIdSize + OffsetSize + SizeSize
	TimestampSize      = 8 // int64 size
	NeedlePaddingSize  = 8
	CookieSize         = 4
)

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func (o *StorageObject) prepareWriteBuffer(writeBytes *bytes.Buffer) (int64, error) {
	writeBytes.Reset()

	header := make([]byte, NeedleHeaderSize+TimestampSize)
	if len(o.Name) >= math.MaxUint8 {
		o.NameSize = math.MaxInt8
	} else {
		o.NameSize = uint8(len(o.Name))
	}
	o.DataSize, o.MimeSize = uint64(len(o.Data)), uint8(len(o.Mime))

	writeBytes.Write(header[0:NeedleHeaderSize])
	if o.DataSize > 0 {
		writeBytes.Write(header[0:4])
		writeBytes.Write(o.Data)
		writeBytes.Write(header[0:1])
	}
	// TODO

	return int64(o.DataSize), nil
}

func (o *StorageObject) Append(offset uint64, size uint64) {

}

func (o *StorageObject) ReadBytes(bytes []byte, offset uint64, size uint64) error {
	return nil
}

//func (o *StorageObject) ReadData(offset int64, size finder.Size) error {
//	bytes, err := ReadStorageObjectBlob(o, offset, size)
//}
//
//func ReadStorageObjectBlob(o *StorageObject, offset int64, size finder.Size) ([]byte, error) {
//	dataSlice := make([]byte, size)
//
//	var n int
//	n, err := r.readAt
//}
