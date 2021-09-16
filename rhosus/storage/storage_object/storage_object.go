package so

import "parasource/rhosus/rhosus/storage/finder"

type StorageObject struct {
	Id   string
	Size finder.Size

	DataSize uint32
	Data     []byte
	Flags    byte
	NameSize uint8
	Name     []byte
	MimeSize uint8
	Mime     []byte
	MetaSize uint16
	Meta     []byte

	Checksum CRC
	Padding  []byte
}

func createStorageObjectFromRequest() {

}

func (s *StorageObject) ParsePath(fid string) error {
	return nil
}
