package so

import (
	"bytes"
	"github.com/parasource/rhosus/rhosus/storage/finder"
	utilbytes "github.com/parasource/rhosus/rhosus/util/bytes"
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

func (o *StorageObject) prepareWriteBuffer(writeBytes *bytes.Buffer) (finder.Size, error) {
	writeBytes.Reset()

	header := make([]byte, NeedleHeaderSize+TimestampSize)
	if len(o.Name) >= math.MaxUint8 {
		o.NameSize = math.MaxInt8
	} else {
		o.NameSize = uint8(len(o.Name))
	}
	o.DataSize, o.MimeSize = uint32(len(o.Data)), uint8(len(o.Mime))

	writeBytes.Write(header[0:NeedleHeaderSize])
	if o.DataSize > 0 {
		utilbytes.Uint32toBytes(header[0:4], o.DataSize)
		writeBytes.Write(header[0:4])
		writeBytes.Write(o.Data)
		writeBytes.Write(header[0:1])
	}
	// TODO

	return finder.Size(o.DataSize), nil
}

func (o *StorageObject) Append(offset uint64, size finder.Size) {

}

func (o *StorageObject) ReadBytes(bytes []byte, offset uint64, size finder.Size) error {
	return nil
}
