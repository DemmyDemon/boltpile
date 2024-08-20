package storage

import (
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
)

func DeleteExpiredFile(pile, entry string) error {
	return errors.New("not implemented")
}

func VoidExpired(config Config, db *bbolt.DB) {
	now := time.Now()
	err := db.Update(func(tx *bbolt.Tx) error {
		for pile, data := range config.Piles {
			debug := log.Debug().Str("pile", pile).Str("operation", "expire")
			if data.Lifetime.Seconds() > 0 {
				debug = debug.Str("lifetime", data.Lifetime.String())
				bucket := tx.Bucket([]byte(pile))
				if bucket == nil {
					return fmt.Errorf("Pile %s does not have a bucket", pile)
				}
				debug = debug.Int("keys", bucket.Stats().KeyN)
				expired := []string{}
				bucket.ForEach(func(k, v []byte) error {
					entry := string(k)
					timestamp, err := time.Parse(TIME_FORMAT, string(v))
					if err != nil {
						return fmt.Errorf("parsing pile %s entry %s time: %w", pile, entry, err)
					}
					expires := timestamp.Add(data.Lifetime.Duration)
					if now.After(expires) {
						expired = append(expired, entry)
					}
					return nil
				})
				for _, entry := range expired {
					log.Info().Str("operation", "expire").Str("pile", pile).Str("entry", entry).Msg("Expired!")
				}
				debug.Int("expired", len(expired)).Msg("OK")
			} else {
				debug.Msg("Entries do not expire")
			}
		}
		return nil
	})
	if err != nil {
		log.Error().Err(err).Msg("Error during VoidExpire operation")
	}
}
