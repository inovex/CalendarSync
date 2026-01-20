package transformation

import (
	"testing"

	"github.com/inovex/CalendarSync/internal/models"
	"github.com/stretchr/testify/assert"
)

// verify transformer SetVisibility
func TestSetVisibility_Transform(t *testing.T) {
	tt := []struct {
		name               string
		visibility         string
		expectedVisibility string
	}{
		{
			name:               "set visibility to private",
			visibility:         "private",
			expectedVisibility: "private",
		},
		{
			name:               "set visibility to public",
			visibility:         "public",
			expectedVisibility: "public",
		},
		{
			name:               "set visibility to confidential",
			visibility:         "confidential",
			expectedVisibility: "confidential",
		},
		{
			name:               "set visibility to default",
			visibility:         "default",
			expectedVisibility: "default",
		},
		{
			name:               "invalid visibility is ignored",
			visibility:         "invalid",
			expectedVisibility: "",
		},
		{
			name:               "empty visibility is ignored",
			visibility:         "",
			expectedVisibility: "",
		},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expectedTitle := "Test Event"

			source := models.Event{
				Title:      expectedTitle,
				Visibility: "default",
			}
			sink := models.Event{
				Title:      expectedTitle,
				Visibility: "",
			}

			transformer := SetVisibility{Visibility: tc.visibility}
			event, err := transformer.Transform(source, sink)

			assert.Nil(t, err)

			expectedEvent := models.Event{
				Title:      expectedTitle,
				Visibility: tc.expectedVisibility,
			}
			assert.Equal(t, expectedEvent, event)
		})
	}
}



