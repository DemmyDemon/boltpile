package storage

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
)

func IsConfiguredBucket(buckets [][]byte, name []byte) bool {
	if buckets == nil {
		return false
	}
	if name == nil {
		return false
	}
	if len(name) == 0 {
		return false
	}
CANDIDATES:
	for _, candidate := range buckets {
		if len(candidate) == len(name) {
			for i, b := range candidate {
				if name[i] != b {
					continue CANDIDATES
				}
			}
			return true
		}
	}
	return false
}

func Startup(config Config, db *bbolt.DB) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucketNames := config.BucketNames()
		if len(bucketNames) == 0 {
			return errors.New("no piles configured")
		}
		for _, bucketName := range bucketNames {
			cfg := config.Piles[string(bucketName)]
			newBucket, err := tx.CreateBucketIfNotExists(bucketName)
			if err != nil {
				return err
			}
			size := newBucket.Stats().KeyN
			log.Info().
				Str("pile", string(bucketName)).
				Int("entries", size).
				Bool("read key", cfg.GETKey != "").
				Bool("write key", cfg.POSTKey != "").
				Str("lifetime", cfg.Lifetime.String()).
				Str("CORS origin", cfg.Origin).
				Msg("Ready!")
		}
		return tx.ForEach(func(name []byte, bucket *bbolt.Bucket) error {
			if !IsConfiguredBucket(bucketNames, name) {
				size := bucket.Stats().KeyN
				log.Warn().Str("pile", string(name)).Int("keys", size).Msg("Not in configuration, so ***REMOVED***")
				err := tx.DeleteBucket(name)
				if err != nil {
					return err
				}
			}
			return nil
		})
	})
}
