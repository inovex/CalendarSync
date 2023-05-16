package transformation

import (
	"github.com/stretchr/testify/assert"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"
	"testing"
)

func TestKeepTitle_Transform(t *testing.T) {
	tt := []struct {
		name          string
		sourceTitle   string
		sinkTitle     string
		expectedTitle string
	}{
		{
			name:          "keep empty title with empty source title and empty sink title",
			sourceTitle:   "",
			sinkTitle:     "",
			expectedTitle: "",
		},
		{
			name:          "keep empty title with empty source title and empty sink title",
			sourceTitle:   "",
			sinkTitle:     "Foo",
			expectedTitle: "",
		},
		{
			name:          "keep title with source title and empty sink title",
			sourceTitle:   "Foo",
			sinkTitle:     "",
			expectedTitle: "Foo",
		},
		{
			name:          "keep title with source title and sink title",
			sourceTitle:   "Foo",
			sinkTitle:     "Bar",
			expectedTitle: "Foo",
		},
		{
			name:          "keep title with source title and same sink title",
			sourceTitle:   "Foo",
			sinkTitle:     "Foo",
			expectedTitle: "Foo",
		},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expectedDescription := "description for testing"

			source := models.Event{
				Title:       tc.sourceTitle,
				Description: "ignore this description",
			}
			sink := models.Event{
				Title:       tc.sinkTitle,
				Description: expectedDescription,
			}

			var transformer KeepTitle
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
