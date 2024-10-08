package storage

import (
	"io"
	"time"
)

const (
	TIME_FORMAT = time.RFC3339
)

type CreateWithFunc func(id string, destination io.Writer) error
type GetWithFunc func(metaData EntryMeta, MIMEType string, file io.Reader) error

type EntryGetter interface {
	GetEntry(pile string, entry string, read GetWithFunc) (err error)
}
type EntryCreator interface {
	CreateEntry(pile string, entry string, creator CreateWithFunc) (entryID string, err error)
}
type EntryHandler interface {
	EntryGetter
	EntryCreator
}
type Starter interface {
	Startup(Config) error
}
type PileGetter interface {
	GetPileEntries(pile string) (map[string]EntryMeta, error)
}
