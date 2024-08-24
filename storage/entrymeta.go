package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

type EntryMeta struct {
	version  uint8
	filename string
	created  time.Time
}

func NewEntryMeta(filename string, created time.Time) EntryMeta {
	return EntryMeta{
		version:  1,
		filename: filename,
		created:  created,
	}
}
func EntryMetaFromBytes(data []byte) (EntryMeta, error) {
	if data == nil || len(data) < 1 {
		return EntryMeta{}, errors.New("empty entry metadata")
	}
	version := uint8(data[0])
	switch version {
	case 1:
		return decodeVersionOne(data)
	default:
		oldstyle := string(data)
		timestamp, err := time.Parse(TIME_FORMAT, oldstyle)
		if err != nil {
			return EntryMeta{}, fmt.Errorf("could not make sense of old-style value %q", oldstyle)
		}
		return EntryMeta{
			version:  1,
			created:  timestamp,
			filename: "data",
		}, nil
	}
}

func (em EntryMeta) Bytes() ([]byte, error) {
	return encodeVersionOne(em)
}

func (em EntryMeta) String() string {
	return em.filename
}
func (em EntryMeta) Filename() string {
	return em.filename
}
func (em EntryMeta) Time() time.Time {
	return em.created
}
func (em EntryMeta) IsZero() bool {
	return em.filename == ""
}

func encodeVersionOne(em EntryMeta) ([]byte, error) {
	data := make([]byte, 0, 24)
	data, err := binary.Append(data, binary.LittleEndian, em.version)
	if err != nil {
		return data, err
	}
	data, err = binary.Append(data, binary.LittleEndian, em.created.Unix())
	if err != nil {
		return data, err
	}
	return append(data, []byte(em.filename)...), nil
}

func decodeVersionOne(data []byte) (EntryMeta, error) {
	entry := EntryMeta{
		version: uint8(data[0]),
	}

	timestamp := int64(0)
	_, err := binary.Decode(data[1:9], binary.LittleEndian, &timestamp)
	if err != nil {
		return entry, fmt.Errorf("decoding timestamp: %w", err)
	}
	entry.created = time.Unix(int64(timestamp), 0)

	entry.filename = string(data[9:])
	return entry, nil
}
