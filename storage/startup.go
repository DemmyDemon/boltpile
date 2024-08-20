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
		buckets := config.BucketNames()
		if len(buckets) == 0 {
			return errors.New("no piles configured")
		}
		for _, bucket := range buckets {
			cfg := config.Piles[string(bucket)]
			newBucket, err := tx.CreateBucketIfNotExists(bucket)
			if err != nil {
				return err
			}
			size := newBucket.Stats().KeyN
			log.Info().Str("pile", string(bucket)).Int("keys", size).Str("lifetime", cfg.Lifetime.String()).Str("CORS origin", cfg.Origin).Msg("Ready!")
		}
		return tx.ForEach(func(name []byte, bucket *bbolt.Bucket) error {
			if !IsConfiguredBucket(buckets, name) {
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
