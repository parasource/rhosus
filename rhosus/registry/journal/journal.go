package journal

type Journal struct {
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
