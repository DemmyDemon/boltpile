package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
					if err := bucket.Delete([]byte(entry)); err != nil {
						return fmt.Errorf("delete expired entry %s in bolt: %w", entry, err)
					}
					path := filepath.Join("piles", pile, entry)
					if err := os.Remove(path); err != nil {
						if !os.IsNotExist(err) {
							return fmt.Errorf("delete expired file %s: %w", entry, err)
						}
						log.Warn().Str("operation", "expire").Str("pile", pile).Str("entry", entry).Msg("Expired file already doesn't exist!")
					}
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

type QuitSignalChan chan<- interface{}

func StartExpireLoop(interval time.Duration, config Config, db *bbolt.DB) QuitSignalChan {

	VoidExpired(config, db)

	ticker := time.NewTicker(interval)
	quit := make(chan interface{})
	go func() {
		for {
			select {
			case <-ticker.C:
				VoidExpired(config, db)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	return quit
}
