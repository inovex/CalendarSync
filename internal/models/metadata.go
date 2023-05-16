package models

import (
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
)

const (
	keyEventID          = "EventID"
	keyOriginalEventUri = "OriginalEventUri"
	keySourceID         = "SourceID"
)

// Errors raised by package models
var (
	ErrMetadataNotFound = errors.New("could not find metadata")
)

// Metadata describes the metadata which is added to events read from the source.
// The data is either calculated from the original event or given in the config file.
type Metadata struct {
	// SyncID is a unique ID which links the original event with the synced copy/copies
	SyncID string `json:"SyncID"`
	// OriginalEventUri is an URI which points to the original event which was synced. This is usually an URL.
	OriginalEventUri string `json:"OriginalEventUri"`
	// SourceID contains the ID of the source which this event was imported from
	SourceID string `json:"SourceID"`
}

func Hash(s string) uint64 {
	h := fnv.New64()
	h.Write([]byte(s))
	return h.Sum64()
}

func NewEventID(seed string) string {
  // We hash the event id, as we need some common denominator for the event IDs
  // We can't use the original event id as the event id in the sink, because the allowed formats differ
  // between the adapters.
	return fmt.Sprint(Hash(seed))
}

func NewEventMetadata(syncId, originalEventUri, sourceID string) *Metadata {
	return &Metadata{
		SyncID:           NewEventID(syncId),
		OriginalEventUri: originalEventUri,
		SourceID:         sourceID,
	}
}

// EventMetadataFromMap creates the Metadata object from a map of strings
// this func validates if the map contains the expected keys. If the keys are not the way we expect,
// we're returing an error of type ErrMetadataNotFound
func EventMetadataFromMap(md map[string]string) (*Metadata, error) {
	var metadata Metadata

	var ok bool
	if metadata.SyncID, ok = md[keyEventID]; !ok {
		return nil, fmt.Errorf("%w: key not exists %s", ErrMetadataNotFound, keyEventID)
	}

	if metadata.OriginalEventUri, ok = md[keyOriginalEventUri]; !ok {
		return nil, fmt.Errorf("%w: key not exists %s", ErrMetadataNotFound, keyOriginalEventUri)
	}

	if metadata.SourceID, ok = md[keySourceID]; !ok {
		return nil, fmt.Errorf("%w: key not exists %s", ErrMetadataNotFound, keySourceID)
	}
	metadata.SourceID = strings.Trim(metadata.SourceID, "\"\\")

	return &metadata, nil
}

// Map returns a map[string]string of the metadata.
// The keys match the Metadata struct field names.
func (m Metadata) Map() map[string]string {
	return map[string]string{
		keyEventID:          m.SyncID,
		keyOriginalEventUri: m.OriginalEventUri,
		keySourceID:         m.SourceID,
	}
}
