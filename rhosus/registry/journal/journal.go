package journal

import (
	master_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/parasource/rhosus/rhosus/util/queue"
)

type Journalist interface {
}

type Journal struct {
	recordsQueue    *queue.Queue
	dispatchRecords func([]*master_pb.Entry)
}

func (j *Journal) WriteToJournal(t string, data []byte) error {
	var err error

	// todo

	return err
}

func (j *Journal) GetAt(at int64) (t string, data []byte, err error) {

	// todo

	return "", nil, err
}

func (j *Journal) GetRange(from int64, to int64) [][]string {

	// todo

	return nil
}
