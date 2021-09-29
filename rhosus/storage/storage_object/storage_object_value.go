package so

import (
	"github.com/google/btree"
)

type StorageObjectValue struct {
	Key    string
	Offset int64
	Size   uint64
}

func (v StorageObjectValue) Less(than btree.Item) bool {
	that := than.(StorageObjectValue)
	return v.Key < that.Key
}

//func (v StorageObjectValue) toBytes() []byte {
//	return
//}
//
//func ToBytes(key string, offset finder.Offset, size finder.Size) []byte {
//	bytes := make([]byte, NeedleIdSize+OffsetSize+SizeSize)
//	NeedleIdToBytes(bytes[0:NeedleIdSize], key)
//	OffsetToBytes(bytes[NeedleIdSize:NeedleIdSize+OffsetSize], offset)
//	util.Uint32toBytes(bytes[NeedleIdSize+OffsetSize:NeedleIdSize+OffsetSize+SizeSize], uint32(size))
//	return bytes
//}