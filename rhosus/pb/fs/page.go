package fs

func (p *Page) AppendBlocks(blocks map[string]*Block) error {
	var err error

	for uid, block := range blocks {
		p.Blocks[uid] = block
	}

	return err
}

func (p *Page) HasFreeSpace() bool {
	return true
}
