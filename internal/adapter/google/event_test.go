package google

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/inovex/CalendarSync/internal/models"
	"google.golang.org/api/calendar/v3"
)

func Test_ensureMetadata(t *testing.T) {
	adapterSourceID := "testSourceId"
	tt := []struct {
		name             string
		event            calendar.Event
		expectedMetadata *models.Metadata
	}{
		{
			name:             "empty input event",
			event:            calendar.Event{},
			expectedMetadata: models.NewEventMetadata("", "", adapterSourceID),
		},
		{
			name: "complete input event with metadata",
			event: calendar.Event{
				ExtendedProperties: &calendar.EventExtendedProperties{
					Private: map[string]string{
						"EventID":          "test",
						"OriginalEventUri": "test",
						"ContentHash":      "test",
						"SourceID":         "test",
					},
				},
			},
			expectedMetadata: &models.Metadata{
				SyncID:           "test",
				OriginalEventUri: "test",
				SourceID:         "test",
			},
		},
		{
			name: "missing event property EventID",
			event: calendar.Event{
				Id:       "eventID",
				HtmlLink: "htmlLink",
				Etag:     "eTag",
				ExtendedProperties: &calendar.EventExtendedProperties{
					Private: map[string]string{
						"OriginalEventUri": "test",
						"ContentHash":      "test",
						"SourceID":         "test",
					},
				},
			},
			expectedMetadata: models.NewEventMetadata("eventID", "htmlLink", adapterSourceID),
		},
		{
			name: "missing event property OriginalEventUri",
			event: calendar.Event{
				Id:       "eventID",
				HtmlLink: "htmlLink",
				Etag:     "eTag",
				ExtendedProperties: &calendar.EventExtendedProperties{
					Private: map[string]string{
						"SyncID":      "test",
						"ContentHash": "test",
						"SourceID":    "test",
					},
				},
			},
			expectedMetadata: models.NewEventMetadata("eventID", "htmlLink", adapterSourceID),
		},
		{
			name: "missing event property ContentHash",
			event: calendar.Event{
				Id:       "eventID",
				HtmlLink: "htmlLink",
				Etag:     "eTag",
				ExtendedProperties: &calendar.EventExtendedProperties{
					Private: map[string]string{
						"SyncID":           "test",
						"OriginalEventUri": "test",
						"SourceID":         "test",
					},
				},
			},
			expectedMetadata: models.NewEventMetadata("eventID", "htmlLink", adapterSourceID),
		},
		{
			name: "missing event property SourceID",
			event: calendar.Event{
				Id:       "eventID",
				HtmlLink: "htmlLink",
				Etag:     "eTag",
				ExtendedProperties: &calendar.EventExtendedProperties{
					Private: map[string]string{
						"SyncID":           "test",
						"OriginalEventUri": "test",
						"ContentHash":      "test",
					},
				},
			},
			expectedMetadata: models.NewEventMetadata("eventID", "htmlLink", adapterSourceID),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			metadata := ensureMetadata(&tc.event, adapterSourceID)

			assert.Equal(t, tc.expectedMetadata, metadata)
		})
	}
}
