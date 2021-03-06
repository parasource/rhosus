package data

import (
	"github.com/parasource/rhosus/rhosus/pb/fs_pb"
	"github.com/parasource/rhosus/rhosus/util/uuid"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestManager_WriteBlocks(t *testing.T) {

	m := &Manager{

		shutdown:         false,
		isReceivingPages: false,
	}

	pmap, err := NewPartitionsMap(defaultPartitionsDir, 1024)
	assert.Nil(t, err)
	m.parts = pmap
	pmap.minPartitionsCount = 1

	var data []byte
	for i := 0; i < 15*1024*1024; i++ {
		data = append(data, byte('a'))
	}
	blocks := make(map[string]*fs_pb.Block, 256)
	for i := 0; i < 128; i++ {
		uid, _ := uuid.NewV4()

		blocks[uid.String()] = &fs_pb.Block{
			Id:     uid.String(),
			FileId: "123123",
			Len:    64,
			Data:   data,
		}
	}
	//_, err = m.WriteBlocks(blocks)
	assert.Nil(t, err)
}
