package so

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/util"
	"hash/crc32"
	"io"
)

var table = crc32.MakeTable(crc32.Castagnoli)

type CRC uint32

func NewCRC(b []byte) CRC {
	return CRC(0).Update(b)
}

func (c CRC) Update(b []byte) CRC {
	return CRC(crc32.Update(uint32(c), table, b))
}

func (c CRC) Value() uint32 {
	return uint32(c>>15|c<<17) + 0xa282ead8
}

func (o *StorageObject) Etag() string {
	bits := make([]byte, 4)
	util.Uint32toBytes(bits, uint32(o.Checksum))
	return fmt.Sprintf("%x", bits)
}

func NewCRCwriter(w io.Writer) *CRCwriter {

	return &CRCwriter{
		crc: CRC(0),
		w:   w,
	}

}

type CRCwriter struct {
	crc CRC
	w   io.Writer
}

func (c *CRCwriter) Write(p []byte) (n int, err error) {
	n, err = c.w.Write(p) // with each write ...
	c.crc = c.crc.Update(p)
	return
}

func (c *CRCwriter) Sum() uint32 { return c.crc.Value() } // final hash
