package storage

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
)

const (
	MAX_SIZE_MB     = 5
	ACCESS_DENIED   = `{"error":"access denied"}`
	ENTRY_NOT_FOUND = `{"error":"entry not found"}`
	ENTRY_TOO_BIG   = `{"error":"entry too big"}`
	OOOPS           = `{"error":"we messed up on our end"}`
	SUCCESS         = `{"success":true, "size":%d, "entry":%q}`
)

func SendMessage(w http.ResponseWriter, statusCode int, messge string) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(messge))
}

func GetFile(db *bbolt.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pile := r.PathValue("pile")
		entry := r.PathValue("entry")
		err := db.View(func(tx *bbolt.Tx) error {
			// TODO:  RemoteAddr isn't adjusted for proxying.
			logEntry := log.Info().Str("operation", "read").Str("pile", pile).Str("entry", entry).Str("peer", r.RemoteAddr)

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
			// TODO: Check that this is okay to serve from the metadata
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
			file.Seek(0, 0)
			MIMEType := http.DetectContentType(buf[:read])
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

func PutFile(db *bbolt.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pile := r.PathValue("pile")
		// TODO:  RemoteAddr isn't adjusted for proxying.
		logEntry := log.Info().Str("operation", "write").Str("pile", pile).Str("peer", r.RemoteAddr)

		r.Body = http.MaxBytesReader(w, r.Body, MAX_SIZE_MB<<20+512)
		err := r.ParseMultipartForm(MAX_SIZE_MB << 20)
		if err != nil {
			logEntry.Err(err).Msg("Oversize file?")
			SendMessage(w, http.StatusBadRequest, ENTRY_TOO_BIG)
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
				SendMessage(w, http.StatusInternalServerError, OOOPS)
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

			err = bucket.Put([]byte(id.String()), []byte("Okay, whatever"))
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
