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

			case dataOpStoreFiles:

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

			case dataOpGetFilesBatch:

				fileIDs := reqs[i].args[0].([]string)
				var res []string

				err := s.db.View(func(tx *bolt.Tx) error {

					b := tx.Bucket([]byte(filesStorageBucketName))
					for _, id := range fileIDs {
						res = append(res, string(b.Get([]byte(id))))
					}

					return nil
				})

				reqs[i].done(res, err)

			case dataOpDeleteFiles:

				fileIDs := reqs[i].args[0].([]string)

				err := s.db.Update(func(tx *bolt.Tx) error {
					var err error

					b := tx.Bucket([]byte(filesStorageBucketName))
					for _, id := range fileIDs {
						err = b.Delete([]byte(id))
					}

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

			case dataOpStoreBatchBlocks:

				data := reqs[i].args[0].(map[string]string)

				err := s.db.Batch(func(tx *bolt.Tx) error {
					var err error

					for blockID, partitionID := range data {
						b := tx.Bucket([]byte(blocksStorageBucketName))

						err = b.Put([]byte(blockID), []byte(partitionID))
					}

					return err
				})

				reqs[i].done(nil, err)

			case dataOpGetBlocks:

				blockIDs := reqs[i].args[0].([]string)
				var res map[string]string

				err := s.db.View(func(tx *bolt.Tx) error {
					var err error

					b := tx.Bucket([]byte(blocksStorageBucketName))
					for _, id := range blockIDs {
						res[id] = string(b.Get([]byte(id)))
					}

					return err
				})

				reqs[i].done(res, err)

			case dataOpDeleteBlocks:

				blockIDs := reqs[i].args[0].([]string)

				err := s.db.Update(func(tx *bolt.Tx) error {
					var err error

					b := tx.Bucket([]byte(blocksStorageBucketName))
					for _, id := range blockIDs {
						err = b.Delete([]byte(id))
					}

					return err
				})

				reqs[i].done(nil, err)

			}
		}

		reqs = nil
	}
}
