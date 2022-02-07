package backend

import (
	bolt "go.etcd.io/bbolt"
)

func (s *Storage) loopFileHandlers() {

	var reqs []StoreReq

	for {

		select {
		case <-s.NotifyShutdown():
			return
		case r := <-s.fileReqC:
			reqs = append(reqs, r)
		}

	loop:
		for len(reqs) < 512 {
			select {
			case <-s.NotifyShutdown():
				return
			case sr := <-s.fileReqC:
				reqs = append(reqs, sr)
			default:
				break loop
			}
		}

		for i := range reqs {
			switch reqs[i].op {

			// ------------------
			// File handlers
			// ------------------
			case dataOpStoreFile:

				path := reqs[i].args[0].(string)
				data := reqs[i].args[1].(string)

				err := s.db.Update(func(tx *bolt.Tx) error {
					var err error

					b := tx.Bucket([]byte(filesStorageBucketName))
					err = b.Put([]byte(path), []byte(data))

					return err
				})

				reqs[i].done(nil, err)

			case dataOpStoreBatch:

				batch := reqs[i].args[0].(map[string]string)

				err := s.db.Batch(func(tx *bolt.Tx) error {
					var err error

					b := tx.Bucket([]byte(filesStorageBucketName))
					for path, data := range batch {
						err = b.Put([]byte(path), []byte(data))
					}

					return err
				})

				reqs[i].done(nil, err)

			case dataOpGetFile:

				path := reqs[i].args[0].(string)
				var res []byte

				err := s.db.View(func(tx *bolt.Tx) error {

					b := tx.Bucket([]byte(filesStorageBucketName))
					res = b.Get([]byte(path))

					return nil
				})

				reqs[i].done(string(res), err)

			case dataOpDeleteFile:

				path := reqs[i].args[0].(string)

				err := s.db.Update(func(tx *bolt.Tx) error {
					var err error

					b := tx.Bucket([]byte(filesStorageBucketName))
					err = b.Delete([]byte(path))

					return err
				})

				reqs[i].done(nil, err)

			}
		}

		reqs = nil
	}
}

func (s *Storage) loopBlocksHandlers() {

	var reqs []StoreReq

	for {

		select {
		case <-s.NotifyShutdown():
			return
		case r := <-s.blocksReqC:
			reqs = append(reqs, r)
		}

	loop:
		for len(reqs) < 512 {
			select {
			case <-s.NotifyShutdown():
				return
			case sr := <-s.blocksReqC:
				reqs = append(reqs, sr)
			default:
				break loop
			}
		}

		for i := range reqs {
			switch reqs[i].op {

			// ------------------
			// File handlers
			// ------------------
			case dataOpStoreBlocks:

				fileId := reqs[i].args[0].(string)
				blocks := reqs[i].args[1].(string)

				err := s.db.Update(func(tx *bolt.Tx) error {
					var err error

					b := tx.Bucket([]byte(blocksStorageBucketName))

					err = b.Put([]byte(fileId), []byte(blocks))

					return err
				})

				reqs[i].done(nil, err)

			case dataOpStoreBatchBlocks:

				data := reqs[i].args[0].(map[string]string)

				err := s.db.Batch(func(tx *bolt.Tx) error {
					var err error

					for fileID, fBlocks := range data {
						b := tx.Bucket([]byte(blocksStorageBucketName))

						err = b.Put([]byte(fileID), []byte(fBlocks))
					}

					return err
				})

				reqs[i].done(nil, err)

			case dataOpGetBlocks:

				fileId := reqs[i].args[0].(string)
				var res []byte

				err := s.db.View(func(tx *bolt.Tx) error {
					var err error

					b := tx.Bucket([]byte(blocksStorageBucketName))
					res = b.Get([]byte(fileId))

					return err
				})

				reqs[i].done(string(res), err)

			case dataOpDeleteBlocks:

				fileId := reqs[i].args[0].(string)

				err := s.db.Update(func(tx *bolt.Tx) error {

					b := tx.Bucket([]byte(blocksStorageBucketName))
					err := b.Delete([]byte(fileId))

					return err
				})

				reqs[i].done(nil, err)

			}
		}

		reqs = nil
	}
}
