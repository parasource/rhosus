package so

type StorageObject struct {
	Id   string
	Size uint64

	Cookie   string
	DataSize uint64
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
