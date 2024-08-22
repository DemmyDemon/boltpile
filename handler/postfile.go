package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/DemmyDemon/boltpile/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
)

func PostFile(db *bbolt.DB, config storage.Config, limiter *RateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pile := r.PathValue("pile")
		peer := DeterminePeer(config, r)

		if !limiter.Allow(peer) {
			log.Warn().Str("operation", "write").Str("pile", pile).Str("peer", peer).Msg("Hit the rate limit!")
			SendMessage(w, http.StatusTooManyRequests, CHILL_OUT)
			return
		}

		logEntry := log.Info().Str("operation", "write").Str("pile", pile).Str("peer", peer)

		pileConfig, err := config.Pile(pile)
		if err != nil {
			log.Error().Err(err).Str("operation", "write").Str("pile", pile).Str("peer", peer).Msg("Couldn't obtain pile config")
			SendMessage(w, http.StatusInternalServerError, OOOPS)
			return
		}
		if !HasBearerToken(pileConfig.POSTKey, r) {
			logEntry.Msg("Invalid or missing bearer token")
			SendMessage(w, http.StatusForbidden, ACCESS_DENIED)
			return
		}

		w.Header().Add("Access-Control-Allow-Origin", pileConfig.Origin)

		r.Body = http.MaxBytesReader(w, r.Body, MAX_SIZE_MB<<20+512)
		err = r.ParseMultipartForm(MAX_SIZE_MB << 20)
		if err != nil {
			logEntry.Err(err).Msg("Error parsing multipart form. Oversize file?")
			SendMessage(w, http.StatusBadRequest, REQUEST_WEIRD)
			return
		}
		err = db.Update(func(tx *bbolt.Tx) error {
			bucket := tx.Bucket([]byte(pile))
			if bucket == nil {
				logEntry.Msg("pile does not exist")
				SendMessage(w, http.StatusForbidden, ACCESS_DENIED)
				return nil
			}

			if err := os.MkdirAll(path.Join("piles", pile), os.ModePerm); err != nil {
				logEntry.Msg("Failed to create pile directory")
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				return nil
			}

			id, err := uuid.NewRandom()
			if err != nil {
				logEntry.Err(err).Msg("Failed to generate a UUID, somehow")
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				return nil
			}
			logEntry = logEntry.Str("entry", id.String())

			file, _, err := r.FormFile("data")
			if err != nil {
				logEntry.Err(err).Msg("Failed to open file reader")
				SendMessage(w, http.StatusBadRequest, REQUEST_WEIRD)
				return nil
			}
			defer file.Close()

			dst, err := os.Create(path.Join("piles", pile, id.String()))
			if err != nil {
				logEntry.Err(err).Msg("Failed to create file")
				return nil
			}
			defer dst.Close()
			size, err := io.Copy(dst, file)
			if err != nil {
				logEntry.Err(err).Msg("Failed to copy data to file")
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				return nil
			}
			now := time.Now().UTC().Format(storage.TIME_FORMAT)
			err = bucket.Put([]byte(id.String()), []byte(now))
			if err != nil {
				logEntry.Err(err).Msg("Could not store file metadata")
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				return nil
			}

			SendMessage(w, http.StatusOK, fmt.Sprintf(SUCCESS, size, id))
			logEntry.Msg("All done! Stored!")

			return nil
		})
		if err != nil {
			log.Error().Err(err).Msg("Transaction failed")
		}
	}
}
