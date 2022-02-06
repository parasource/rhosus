package backend

import bolt "go.etcd.io/bbolt"

func (s *Storage) loopBlocksHandlers() {

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
			case dataOpStoreBlocks:

				path := reqs[i].args[0].(string)
				data := reqs[i].args[1].(string)

				err := s.db.Update(func(tx *bolt.Tx) error {
					var err error

					b := tx.Bucket([]byte(filesStorageBucketName))
					err = b.Put([]byte(path), []byte(data))

					return err
				})

				reqs[i].done(nil, err)

			case dataOpGetBlocks:

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

			case dataOpDeleteBlocks:

				path := reqs[i].args[0].(string)
				var res []byte

				err := s.db.View(func(tx *bolt.Tx) error {

					b := tx.Bucket([]byte(filesStorageBucketName))
					res = b.Get([]byte(path))

					return nil
				})

				reqs[i].done(res, err)

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
