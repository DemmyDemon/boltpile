package storage

import (
	"net/http"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"
)

type BoltDatabase struct {
	db *bbolt.DB
}

func MustOpenBoltDatabase(filename string) BoltDatabase {
	db, err := bbolt.Open(filename, 0600, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Could not open bbolt file")
	}
	return BoltDatabase{db: db}
}
func (eh BoltDatabase) GetEntry(pile string, entry string, get GetWithFunc) error {
	err := eh.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(pile))
		if bucket == nil {
			return ErrNoSuchPile{pile}
		}
		value := bucket.Get([]byte(entry))
		if value == nil {
			return ErrNoSuchEntry{Pile: pile, Entry: entry}
		}
		created, err := time.Parse(TIME_FORMAT, string(value))
		if err != nil {
			return ErrUnparsableTime{Raw: value, ParseError: err}
		}

		file, err := os.Open(path.Join("piles", pile, entry))
		if err != nil {
			if os.IsNotExist(err) {
				return ErrNoSuchEntry{Pile: pile, Entry: entry}
			}
			return ErrDuringFileOperation{Pile: pile, Entry: entry, UpstreamError: err}
		}

		buf := make([]byte, 512)
		read, err := file.Read(buf)
		if err != nil {
			return ErrDuringFileOperation{Pile: pile, Entry: entry, UpstreamError: err}
		}
		MIMEType := http.DetectContentType(buf[:read])
		file.Seek(0, 0)

		err = get(created, MIMEType, file)
		if err != nil {
			return ErrDuringFileOperation{Pile: pile, Entry: entry, UpstreamError: err}
		}
		err = file.Close()
		if err != nil {
			return ErrDuringFileOperation{Pile: pile, Entry: entry, UpstreamError: err}
		}
		return nil
	})
	return err
}
func (eh BoltDatabase) CreateEntry(pile string, entry string, create CreateWithFunc) (string, error) {
	if entry == "" {
		id, err := uuid.NewRandom()
		if err != nil {
			return "", ErrFailedMakingId{err}
		}
		entry = id.String()
	}

	return entry, eh.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(pile))
		if bucket == nil {
			return ErrNoSuchPile{pile}
		}
		if err := os.MkdirAll(path.Join("piles", pile), os.ModePerm); err != nil {
			return ErrFailedCreatingPileDirectory{Pile: pile, UpstreamError: err}
		}
		dstFile, err := os.Create(path.Join("piles", pile, entry))
		if err != nil {
			return ErrFailedCreatingEntryFile{Pile: pile, Entry: entry, UpstreamError: err}
		}
		err = create(entry, dstFile)
		if err != nil {
			dstFile.Close() // ... and hope for the best, I guess XD
			return ErrDuringFileOperation{Pile: pile, Entry: entry, UpstreamError: err}
		}

		if err := dstFile.Close(); err != nil {
			return ErrFailedCreatingEntryFile{Pile: pile, Entry: entry, UpstreamError: err}
		}

		now := time.Now().UTC().Format(TIME_FORMAT)
		if err := bucket.Put([]byte(entry), []byte(now)); err != nil {
			return ErrFailedStoringEntryMetadata{Pile: pile, Entry: entry, UpstreamError: err}
		}

		return nil
	})
}
func (eh BoltDatabase) Startup(config Config) error {
	err := Startup(config, eh.db)
	if err != nil {
		return err
	}
	StartExpireLoop(5*time.Minute, config, eh.db)
	return nil
}
