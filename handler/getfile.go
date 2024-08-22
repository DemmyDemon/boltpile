package handler

import (
	"io"
	"net/http"
	"os"
	"path"
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

		createTime, err := eg.GetEntry(pile, entry)
		if err != nil {
			errLog := log.Error().Err(err).Str("operation", "read").Str("pile", pile).Str("entry", entry).Str("peer", peer)
			switch err.(type) {
			case storage.ErrNoSuchPile:
				SendMessage(w, http.StatusNotFound, ENTRY_NOT_FOUND)
				errLog.Msg("Pile not found")
			case storage.ErrNoSuchEntry:
				SendMessage(w, http.StatusNotFound, ENTRY_NOT_FOUND)
				errLog.Msg("Entry not found in bucket")
			case storage.ErrUnparsableTime:
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				errLog.Err(err).Msg("Failed to parse creation time")
			default:
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				errLog.Err(err).Msg("Unknown error")
			}
			return
		}

		now := time.Now()

		expires := createTime.Add(pileConfig.Lifetime.Duration)
		if now.After(expires) {
			SendMessage(w, http.StatusNotFound, ENTRY_NOT_FOUND)
			logEntry.Msg("Entry has expired, but was not culled yet")
			return
		}

		w.Header().Set("Expires", expires.UTC().Format(http.TimeFormat))
		w.Header().Set("Last-Modified", createTime.UTC().Format(http.TimeFormat))

		file, err := os.Open(path.Join("piles", pile, entry))
		if err != nil {
			if os.IsNotExist(err) {
				SendMessage(w, http.StatusNotFound, ENTRY_NOT_FOUND)
				logEntry.Msg("File not found")
				return
			}

			SendMessage(w, http.StatusInternalServerError, OOOPS)
			logEntry.Err(err).Msg("Error during file open!")
			return
		}
		defer file.Close()
		buf := make([]byte, 512)
		read, err := file.Read(buf)
		if err != nil {
			SendMessage(w, http.StatusInternalServerError, OOOPS)
			logEntry.Err(err).Msg("Error during sniff")
			return
		}
		MIMEType := http.DetectContentType(buf[:read])
		file.Seek(0, 0)

		w.Header().Set("Content-Type", MIMEType)
		w.WriteHeader(http.StatusOK)
		logEntry.Msg("Serving data!")

		_, err = io.Copy(w, file)
		if err != nil {
			log.Error().Err(err).Str("operation", "read").Str("pile", pile).Str("entry", entry).Str("peer", peer).Msg("Error while sending data!")
		}
	}
}
