package transformation

import (
	"github.com/stretchr/testify/assert"
	"github.com/inovex/CalendarSync/internal/models"
	"testing"
)

func TestKeepLocation_Transform(t *testing.T) {
	tt := []struct {
		name             string
		sourceLocation   string
		sinkLocation     string
		expectedLocation string
	}{
		{
			name:             "keep empty location with empty source and empty sink",
			sourceLocation:   "",
			sinkLocation:     "",
			expectedLocation: "",
		},
		{
			name:             "keep location with source and empty sink",
			sourceLocation:   "wonderland",
			sinkLocation:     "",
			expectedLocation: "wonderland",
		},
		{
			name:             "keep empty location with empty source and sink",
			sourceLocation:   "",
			sinkLocation:     "wonderland",
			expectedLocation: "",
		},
		{
			name:             "keep location with source and sink",
			sourceLocation:   "wonderland",
			sinkLocation:     "neverland",
			expectedLocation: "wonderland",
		},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expectedTitle := "title for testing"

			source := models.Event{
				Title:    "ignore this title",
				Location: tc.sourceLocation,
			}
			sink := models.Event{
				Title:    expectedTitle,
				Location: tc.sinkLocation,
			}

			var transfomer KeepLocation
			event, err := transfomer.Transform(source, sink)

			assert.NoError(t, err)
			expectedEvent := models.Event{
				Title:    expectedTitle,
				Location: tc.expectedLocation,
			}
			assert.Equal(t, expectedEvent, event)
		})
	}
}
