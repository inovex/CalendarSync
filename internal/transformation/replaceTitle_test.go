package transformation

import (
	"github.com/stretchr/testify/assert"
	"github.com/inovex/CalendarSync/internal/models"
	"testing"
)

// verify transformer ReplaceTitle
func TestReplaceTitle_Transform(t *testing.T) {
	tt := []struct {
		name          string
		title         string
		expectedTitle string
	}{
		{
			name:          "replace title with empty title and empty value",
			title:         "",
			expectedTitle: "",
		},
		{
			name:          "replace title with empty title and value",
			title:         "",
			expectedTitle: "Foo",
		},
		{
			name:          "replace title with empty value",
			title:         "Foo",
			expectedTitle: "",
		},
		{
			name:          "replace title with value",
			title:         "Foo",
			expectedTitle: "bar",
		},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expectedDescription := "description for testing"

			source := models.Event{
				Title:       tc.title,
				Description: "ignore this description",
			}
			sink := models.Event{
				Title:       tc.expectedTitle,
				Description: expectedDescription,
			}

			transformer := ReplaceTitle{NewTitle: tc.expectedTitle}
			event, err := transformer.Transform(source, sink)

			assert.Nil(t, err)

			expectedEvent := models.Event{
				Title:       tc.expectedTitle,
				Description: expectedDescription,
			}
			assert.Equal(t, expectedEvent, event)
		})
	}
}
