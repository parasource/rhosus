package data

import (
	"fmt"
	"github.com/parasource/rhosus/rhosus/pb/fs"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/sirupsen/logrus"
	"log"
	"os"
	"sync"
)

type Page struct {
	Uid      string
	mu       sync.RWMutex
	Capacity uint64
	Used     uint64
	CursorAt int64
	FilePath string
}

func NewPage() (*Page, error) {
	uid, _ := uuid.NewV4()

	path := fmt.Sprintf("~/Rhosus/datafiles/%v.data", uid.String())
	f, err := os.Create(path)
	defer f.Close()

	if err != nil {
		logrus.Errorf("error creating page data file: %v", err)
		return nil, err
	}

	return &Page{
		Uid:      uid.String(),
		Capacity: 1024 * 1024,
		Used:     1024 * 126,
		CursorAt: 0,
		FilePath: path,
	}, nil
}

func (p *Page) AppendBlocks(blocks map[string]*fs.Block) error {
	var err error

	err = p.appendToDataFile([]byte(""))
	if err != nil {
		return err
	}

	return err
}

func (p *Page) appendToDataFile(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	file, err := os.OpenFile(p.FilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer file.Close()

	n, err := file.Write(data)
	if err != nil {
		return err
	}

	p.CursorAt += int64(n)

	return nil
}

func (p *Page) Marshal() ([]byte, error) {
	page := &fs.Page{
		Uid:          p.Uid,
		UsedSpace:    0,
		Blocks:       nil,
		ChecksumType: 0,
		Checksum:     "",
		UpdatedAt:    0,
	}

	bytes, err := page.Marshal()

	return bytes, err
}

func (p *Page) HasFreeSpace() bool {
	return true
}
