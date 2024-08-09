package transformation

import (
	"strings"
	"testing"

	"github.com/inovex/CalendarSync/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestKeepDescription_Transform(t *testing.T) {
	tt := []struct {
		name                string
		sourceDescription   string
		sinkDescription     string
		expectedDescription string
	}{
		{
			name:                "keep empty description with empty source and empty sink",
			sourceDescription:   "",
			sinkDescription:     "",
			expectedDescription: "",
		},
		{
			name:                "keep empty description with empty source and random sink",
			sourceDescription:   "",
			sinkDescription:     "foo",
			expectedDescription: "",
		},
		{
			name:                "keep description with source and empty sink",
			sourceDescription:   "foo",
			sinkDescription:     "",
			expectedDescription: "foo",
		},
		{
			name:                "keep description with source and sink",
			sourceDescription:   "foo",
			sinkDescription:     "bar",
			expectedDescription: "foo",
		},
		{
			name:                "removes html from description",
			sourceDescription:   "<test>Headline</test><h1>Headline2</h1>",
			sinkDescription:     "bar",
			expectedDescription: "Headline<h1>Headline2</h1>",
		},
		{
			name:                "keep html-linebreaks from description",
			sourceDescription:   "<h1>Headline2</h1><br><br/>ohai\r\n",
			sinkDescription:     "bar",
			expectedDescription: "<h1>Headline2</h1><br><br/>ohai",
		},
		{
			name:                "truncates description after 4000 chars",
			sourceDescription:   strings.Repeat("a", 4000),
			sinkDescription:     "bar",
			expectedDescription: strings.Repeat("a", 4000),
		},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expectedTitle := "title for testing"

			source := models.Event{
				Description: tc.sourceDescription,
				Title:       "ignore this title",
			}
			sink := models.Event{
				Description: tc.sinkDescription,
				Title:       expectedTitle,
			}

			var transformer KeepDescription
			event, err := transformer.Transform(source, sink)

			assert.Nil(t, err)

			expectedEvent := models.Event{
				Title:       expectedTitle,
				Description: tc.expectedDescription,
			}
			assert.Equal(t, expectedEvent, event)
		})
	}
}
