package storage_test

import (
	"testing"
	"time"

	"github.com/DemmyDemon/boltpile/storage"
)

func TestEntryMetaEncoding(t *testing.T) {
	now := time.Now().UTC()
	compare := "dummy_filename.jpg"
	meta := storage.NewEntryMeta(compare, now)
	data, err := meta.Bytes()
	if err != nil {
		t.Errorf("encoding: %s", err)
		return
	}
	anotherMeta, err := storage.EntryMetaFromBytes(data)
	if err != nil {
		t.Errorf("decoding: %s", err)
		return
	}
	if now.Unix() != anotherMeta.Time().Unix() {
		t.Errorf("encoded and decoded times do not match (encoded %s, decoded %s)", now.Format(storage.TIME_FORMAT), anotherMeta.Time().Format(storage.TIME_FORMAT))
		return
	}
	if anotherMeta.Filename() != compare {
		t.Errorf("%q != %q", compare, anotherMeta.Filename())
		return
	}
}

func TestEntryMetaDecodeOldValue(t *testing.T) {
	now := time.Now().UTC()
	oldStyleValue := now.Format(storage.TIME_FORMAT)
	meta, err := storage.EntryMetaFromBytes([]byte(oldStyleValue))
	if err != nil {
		t.Errorf("decoding: %s", err)
		return
	}
	if now.Unix() != meta.Time().Unix() {
		t.Errorf("old-style time and decoded time do not match (old-style %s, decoded %s)", oldStyleValue, meta.Time().Format(storage.TIME_FORMAT))
		return
	}
}
