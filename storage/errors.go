package storage

import "fmt"

type ErrNoSuchPile struct {
	Pile string
}

func (err ErrNoSuchPile) Error() string {
	return fmt.Sprintf("%s: no such pile", err.Pile)
}

type ErrNoSuchEntry struct {
	Pile  string
	Entry string
}

func (err ErrNoSuchEntry) Error() string {
	return fmt.Sprintf("%s/%s: no such entry", err.Pile, err.Entry)
}

type ErrUnparsableTime struct {
	Raw        []byte
	ParseError error
}

func (err ErrUnparsableTime) Error() string {
	return fmt.Sprintf("failed to parse timestamp: %s", err.ParseError.Error())
}

type ErrFailedMakingId struct {
	UpstreamError error
}

func (err ErrFailedMakingId) Error() string {
	return fmt.Sprintf("failed to make an entry ID: %s", err.UpstreamError.Error())
}

type ErrFailedCreatingPileDirectory struct {
	Pile          string
	UpstreamError error
}

func (err ErrFailedCreatingPileDirectory) Error() string {
	return fmt.Sprintf("failed to make a directory for pile %s: %s", err.Pile, err.UpstreamError.Error())
}

type ErrFailedCreatingEntryFile struct {
	Pile          string
	Entry         string
	UpstreamError error
}

func (err ErrFailedCreatingEntryFile) Error() string {
	return fmt.Sprintf("failed to make file for entry %s/%s: %s", err.Pile, err.Entry, err.UpstreamError.Error())
}

type ErrDuringWriteOperation struct {
	Pile          string
	Entry         string
	UpstreamError error
}

func (err ErrDuringWriteOperation) Error() string {
	return fmt.Sprintf("something went wrong after handing off %s/%s to to be written to: %s", err.Pile, err.Entry, err.UpstreamError.Error())
}

type ErrFailedStoringEntryMetadata struct {
	Pile          string
	Entry         string
	UpstreamError error
}

func (err ErrFailedStoringEntryMetadata) Error() string {
	return fmt.Sprintf("error putting %s/%s metadata into database: %s", err.Pile, err.Entry, err.UpstreamError)
}
