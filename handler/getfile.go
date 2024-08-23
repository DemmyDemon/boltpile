package handler

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/DemmyDemon/boltpile/storage"
	"github.com/rs/zerolog/log"
)

func GetFile(eg storage.EntryGetter, config storage.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pile := r.PathValue("pile")
		entry := r.PathValue("entry")
		peer := DeterminePeer(config, r)

		pileConfig, err := config.Pile(pile)
		if err != nil {
			log.Error().Err(err).Str("operation", "read").Str("pile", pile).Str("entry", entry).Str("peer", peer).Msg("Couldn't obtain pile config")
			SendMessage(w, http.StatusInternalServerError, OOOPS)
			return
		}

		logEntry := log.Info().Str("operation", "read").Str("pile", pile).Str("entry", entry).Str("peer", peer)
		if !HasBearerToken(pileConfig.GETKey, r) {
			logEntry.Msg("Invalid or missing bearer token")
			SendMessage(w, http.StatusForbidden, ACCESS_DENIED)
			return
		}

		w.Header().Add("Access-Control-Allow-Origin", pileConfig.Origin)

		err = eg.GetEntry(pile, entry, func(createTime time.Time, MIMEType string, file io.Reader) error {
			now := time.Now()
			expires := createTime.Add(pileConfig.Lifetime.Duration)
			if now.After(expires) {
				return errors.New("entry expired, but was not culled yet")
			}
			w.Header().Set("Expires", expires.UTC().Format(http.TimeFormat))
			w.Header().Set("Last-Modified", createTime.UTC().Format(http.TimeFormat))
			w.Header().Set("Content-Type", MIMEType)
			w.WriteHeader(http.StatusOK)
			logEntry.Msg("Serving data!")
			_, err = io.Copy(w, file)
			return err
		})

		if err != nil {
			errLog := log.Error().Err(err).Str("operation", "read").Str("pile", pile).Str("entry", entry).Str("peer", peer)
			switch err.(type) {
			case storage.ErrNoSuchPile:
				SendMessage(w, http.StatusNotFound, ENTRY_NOT_FOUND)
				errLog.Msg("Pile not found")
			case storage.ErrNoSuchEntry:
				SendMessage(w, http.StatusNotFound, ENTRY_NOT_FOUND)
				errLog.Msg("Entry not found")
			case storage.ErrUnparsableTime:
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				errLog.Err(err).Msg("Failed to parse creation time")
			case storage.ErrDuringFileOperation:
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				errLog.Err(err).Msg("File operation failed")
			default:
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				errLog.Err(err).Msg("Other error")
			}
			return
		}
	}
}
