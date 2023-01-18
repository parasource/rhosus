package storage

import (
	control_pb "github.com/parasource/rhosus/rhosus/pb/control"
	"github.com/rs/zerolog/log"
	"sort"
)

const (
	defaultFilesTableName  = "__files"
	defaultBlocksTableName = "__blocks"

	EntryTypeFile  EntryType = "file"
	EntryTypeBlock EntryType = "block"
)

// StoreFile puts file to memory and
// stores it to backend
func (s *Storage) StoreFile(file *control_pb.FileInfo) error {
	err := s.storeFileInMemory(file)
	if err != nil {
		return err
	}

	fileBytes, err := file.Marshal()
	if err != nil {
		return err
	}

	entry := &Entry{
		Key:   file.Path,
		Value: fileBytes,
	}

	// todo correct error handling
	err = s.backend.Put(EntryTypeFile, []*Entry{entry})
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) storeFileInMemory(file *control_pb.FileInfo) error {
	txn := s.db.Txn(true)
	err := txn.Insert(defaultFilesTableName, file)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	return nil
}

func (s *Storage) GetFile(fileID string) (*control_pb.FileInfo, error) {
	txn := s.db.Txn(false)

	raw, err := txn.First(defaultFilesTableName, "id", fileID)
	if err != nil {
		return nil, err
	}

	switch raw.(type) {
	case *control_pb.FileInfo:
		return raw.(*control_pb.FileInfo), nil
	default:
		return nil, nil
	}
}

func (s *Storage) GetFileByPath(path string) (*control_pb.FileInfo, error) {
	txn := s.db.Txn(false)

	raw, err := txn.First(defaultFilesTableName, "path", path)
	if err != nil {
		return nil, err
	}

	switch raw.(type) {
	case *control_pb.FileInfo:
		return raw.(*control_pb.FileInfo), nil
	default:
		return nil, nil
	}
}

func (s *Storage) GetFilesByParentId(id string) ([]*control_pb.FileInfo, error) {
	txn := s.db.Txn(false)

	var files []*control_pb.FileInfo
	res, err := txn.Get(defaultFilesTableName, "parent_id", id)
	if err != nil {
		return nil, err
	}

	for obj := res.Next(); obj != nil; obj = res.Next() {
		files = append(files, obj.(*control_pb.FileInfo))
	}

	return files, nil
}

func (s *Storage) PutBlocks(blocks []*control_pb.BlockInfo) error {
	var err error

	txn := s.db.Txn(true)
	for _, block := range blocks {
		err = txn.Insert(defaultBlocksTableName, block)
		if err != nil {
			txn.Abort()
			return err
		}
	}
	txn.Commit()

	var entries []*Entry
	for _, block := range blocks {
		blockBytes, err := block.Marshal()
		if err != nil {
			log.Error().Err(err).Msg("error marshalling block")
			continue
		}
		entry := &Entry{
			Key:   block.Id,
			Value: blockBytes,
		}
		entries = append(entries, entry)
	}

	return s.backend.Put(EntryTypeBlock, entries)
}

func (s *Storage) storeBlockInMemory(block *control_pb.BlockInfo) error {
	var err error

	txn := s.db.Txn(true)
	err = txn.Insert(defaultBlocksTableName, block)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	return nil
}

func (s *Storage) GetAllFiles() []*control_pb.FileInfo {
	txn := s.db.Txn(false)

	var files []*control_pb.FileInfo
	res, err := txn.Get(defaultFilesTableName, "id")
	if err != nil {
		return nil
	}

	for obj := res.Next(); obj != nil; obj = res.Next() {
		files = append(files, obj.(*control_pb.FileInfo))
	}

	return files
}

func (s *Storage) GetBlocks(fileID string) ([]*control_pb.BlockInfo, error) {
	txn := s.db.Txn(false)

	var blocks []*control_pb.BlockInfo

	res, err := txn.Get(defaultBlocksTableName, "file_id", fileID)
	if err != nil {
		return nil, err
	}
	for obj := res.Next(); obj != nil; obj = res.Next() {
		blocks = append(blocks, obj.(*control_pb.BlockInfo))
	}

	// sort blocks in sequence order
	sort.Slice(blocks, func(i, j int) bool {
		return blocks[i].Index < blocks[j].Index
	})

	return blocks, nil
}

func (s *Storage) DeleteFileWithBlocks(file *control_pb.FileInfo) error {
	txn := s.db.Txn(true)

	var blocks []*control_pb.BlockInfo
	res, err := txn.Get(defaultBlocksTableName, "file_id", file.Id)
	if err != nil {
		return err
	}
	for obj := res.Next(); obj != nil; obj = res.Next() {
		blocks = append(blocks, obj.(*control_pb.BlockInfo))
		err := txn.Delete(defaultBlocksTableName, obj.(*control_pb.BlockInfo))
		if err != nil {
			txn.Abort()
			return err
		}
	}
	err = txn.Delete(defaultFilesTableName, file)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	// todo correct error handling
	err = s.backend.Delete(EntryTypeFile, []string{file.Path})
	if err != nil {
		return err
	}

	var blockIdsToDelete []string
	for _, block := range blocks {
		blockIdsToDelete = append(blockIdsToDelete, block.Id)
	}

	return s.backend.Delete(EntryTypeBlock, blockIdsToDelete)
}

func (s *Storage) DeleteFile(file *control_pb.FileInfo) error {
	txn := s.db.Txn(true)
	err := txn.Delete(defaultFilesTableName, file)
	if err != nil {
		txn.Abort()
		return err
	}
	txn.Commit()

	// todo correct error handling
	err = s.backend.Delete(EntryTypeFile, []string{file.Path})
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) DeleteBlocks(blocks []*control_pb.BlockInfo) error {
	txn := s.db.Txn(true)
	for _, block := range blocks {
		err := txn.Delete(defaultBlocksTableName, block)
		if err != nil {
			txn.Abort()
			return err
		}
	}
	txn.Commit()

	var blockIdsToDelete []string
	for _, block := range blocks {
		blockIdsToDelete = append(blockIdsToDelete, block.Id)
	}

	return s.backend.Delete(EntryTypeBlock, blockIdsToDelete)
}
