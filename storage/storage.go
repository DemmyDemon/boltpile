package storage

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
)

const (
	TIME_FORMAT     = time.RFC3339
	MAX_SIZE_MB     = 5
	ACCESS_DENIED   = `{"error":"access denied", "success":false}`
	ENTRY_NOT_FOUND = `{"error":"entry not found", "success":false}`
	REQUEST_WEIRD   = `{"error":"request too weird", "success":false}`
	OOOPS           = `{"error":"we messed up on our end", "success":false}`
	SUCCESS         = `{"success":true, "size":%d, "entry":%q}`
)

func SendMessage(w http.ResponseWriter, statusCode int, messge string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(messge))
}

func DeterminePeer(config Config, r *http.Request) string {
	remote := r.RemoteAddr
	peer, _, err := net.SplitHostPort(remote)
	if err != nil {
		log.Warn().Err(err).Msg("Splitting host and port from remote address is weird.")
		return remote
	}
	if config.ForwardHeader != "" {
		if forwardHeader := r.Header.Get(config.ForwardHeader); forwardHeader != "" {
			return forwardHeader
		}
	}
	return peer
}

func GetFile(db *bbolt.DB, config Config) http.HandlerFunc {
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
		w.Header().Add("Access-Control-Allow-Origin", pileConfig.Origin)

		err = db.View(func(tx *bbolt.Tx) error {
			logEntry := log.Info().Str("operation", "read").Str("pile", pile).Str("entry", entry).Str("peer", peer)

			bucket := tx.Bucket([]byte(pile))
			if bucket == nil {
				SendMessage(w, http.StatusNotFound, ENTRY_NOT_FOUND)
				logEntry.Msg("Pile not found")
				return nil
			}

			value := bucket.Get([]byte(entry))
			if value == nil {
				SendMessage(w, http.StatusNotFound, ENTRY_NOT_FOUND)
				logEntry.Msg("Entry not found in bucket")
				return nil
			}

			now := time.Now()
			createTime, err := time.Parse(TIME_FORMAT, string(value))
			if err != nil {
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				logEntry.Err(err).Msg("Failed to parse creation time")
				return nil
			}

			expires := createTime.Add(pileConfig.Lifetime.Duration)
			if now.After(expires) {
				SendMessage(w, http.StatusNotFound, ENTRY_NOT_FOUND)
				logEntry.Msg("Entry has expired, but was not culled yet")
				return nil
			}

			w.Header().Set("Expires", expires.UTC().Format(http.TimeFormat))
			w.Header().Set("Last-Modified", createTime.UTC().Format(http.TimeFormat))

			file, err := os.Open(path.Join("piles", pile, entry))
			if err != nil {
				if os.IsNotExist(err) {
					SendMessage(w, http.StatusNotFound, ENTRY_NOT_FOUND)
					logEntry.Msg("File not found")
					return nil
				}

				SendMessage(w, http.StatusInternalServerError, OOOPS)
				logEntry.Err(err).Msg("Error during file open!")
				return nil
			}
			defer file.Close()
			buf := make([]byte, 512)
			read, err := file.Read(buf)
			if err != nil {
				SendMessage(w, http.StatusInternalServerError, OOOPS)
				logEntry.Err(err).Msg("Error during sniff")
				return nil
			}
			MIMEType := http.DetectContentType(buf[:read])
			file.Seek(0, 0)

			w.Header().Set("Content-Type", MIMEType)
			w.WriteHeader(http.StatusOK)
			logEntry.Msg("Serving data!")

			_, err = io.Copy(w, file)
			return err
		})
		if err != nil {
			log.Error().Err(err).Msg("Error during View operation")
		}
	}
}

func PutFile(db *bbolt.DB, config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pile := r.PathValue("pile")
		peer := DeterminePeer(config, r)
		logEntry := log.Info().Str("operation", "write").Str("pile", pile).Str("peer", peer)

		pileConfig, err := config.Pile(pile)
		if err != nil {
			log.Error().Err(err).Str("operation", "write").Str("pile", pile).Str("peer", peer).Msg("Couldn't obtain pile config")
			SendMessage(w, http.StatusInternalServerError, OOOPS)
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
			now := time.Now().UTC().Format(TIME_FORMAT)
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
