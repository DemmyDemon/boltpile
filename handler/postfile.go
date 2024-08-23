package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/DemmyDemon/boltpile/storage"
	"github.com/rs/zerolog/log"
)

func PostFile(ec storage.EntryCreator, config storage.Config, limiter *RateLimiter) http.HandlerFunc {
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

		maxSize := pileConfig.MaxSize
		if maxSize <= 0 {
			maxSize = MAX_SIZE_DEFAULT
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxSize+512)
		err = r.ParseMultipartForm(maxSize)
		if err != nil {
			logEntry.Err(err).Msg("Error parsing multipart form. Oversize file?")
			SendMessage(w, http.StatusBadRequest, REQUEST_WEIRD)
			return
		}

		size := int64(0)

		entryID, err := ec.CreateEntry(pile, "", func(entry string, dst io.Writer) error {
			file, _, err := r.FormFile("data")
			if err != nil {
				return err
			}
			defer file.Close()
			size, err = io.Copy(dst, file)
			return err
		})

		if err != nil {
			errLog := log.Error().Err(err).Str("operation", "write").Str("pile", pile).Str("entry", entryID).Str("peer", peer)
			switch err.(type) {
			case storage.ErrNoSuchPile:
				errLog.Msg("No such pile")
				SendMessage(w, http.StatusForbidden, ACCESS_DENIED)
			case storage.ErrFailedCreatingPileDirectory:
				errLog.Msg("Could not create directory")
				SendMessage(w, http.StatusInternalServerError, OOOPS)
			case storage.ErrFailedMakingId:
				errLog.Msg("Failed generating UUID, somehow")
				SendMessage(w, http.StatusInternalServerError, OOOPS)
			case storage.ErrFailedCreatingEntryFile:
				errLog.Msg("Well, that didn't work...")
				SendMessage(w, http.StatusInternalServerError, OOOPS)
			case storage.ErrDuringFileOperation:
				errLog.Msg("Looks like weird data from client. Oversize?")
				SendMessage(w, http.StatusBadRequest, REQUEST_WEIRD)
			case storage.ErrFailedStoringEntryMetadata:
				errLog.Msg("I love Bolt, but sometimes...")
				SendMessage(w, http.StatusInternalServerError, OOOPS)
			default:
				errLog.Msg("Well, that was unexpected...")
			}
			return
		}

		SendMessage(w, http.StatusOK, fmt.Sprintf(SUCCESS, size, entryID))
		logEntry.Str("entry", entryID).Msg("All done! Stored!")
	}
}
