package data

import (
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIdxFile_Write(t *testing.T) {
	file, err := NewIdxFile(".", "test")
	assert.Nil(t, err)

	err = file.Write(map[int]IdxBlock{
		0: {
			ID:   "f5f43186-4c6f-4358-906d-3929008e991c",
			Size: 64,
		},
		1: {
			ID:   "b92d35bd-c9b3-4057-a9fb-d5246fa16265",
			Size: 43556,
		},
		10: {
			ID:   "16bbdd53-323d-40ad-8999-661ae5c06156",
			Size: 3342,
		},
	})
	assert.Nil(t, err)

	logrus.Info(file.Load())
}
