package transformation

import (
	"github.com/stretchr/testify/assert"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"
	"testing"
)

func TestKeepMeetingLink_Transform(t *testing.T) {
	tt := []struct {
		name                string
		sourceMeetingLink   string
		sourceDescription   string
		sinkDescription     string
		expectedDescription string
	}{
		{
			name:                "keep empty meetingLink with empty source and empty sink description",
			sourceMeetingLink:   "",
			sourceDescription:   "",
			sinkDescription:     "",
			expectedDescription: "",
		},
		{
			name:                "empty meetingLink keep description",
			sourceMeetingLink:   "",
			sourceDescription:   "foo",
			sinkDescription:     "foo",
			expectedDescription: "foo",
		},
		{
			name:                "add meetingLink to empty description",
			sourceMeetingLink:   "https://meetingLink.de",
			sourceDescription:   "",
			sinkDescription:     "",
			expectedDescription: "original meeting link: https://meetingLink.de\n\n############\n",
		},
		{
			name:                "keep description with source and sink",
			sourceMeetingLink:   "https://meetingLink.de",
			sourceDescription:   "foo",
			sinkDescription:     "bar",
			expectedDescription: "original meeting link: https://meetingLink.de\n\n############\nfoo",
		},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expectedTitle := "title for testing"

			source := models.Event{
				Description: tc.sourceDescription,
				MeetingLink: tc.sourceMeetingLink,
				Title:       "ignore this title",
			}
			sink := models.Event{
				Title: expectedTitle,
			}

			var descriptionTransformer KeepDescription
			event, err := descriptionTransformer.Transform(source, sink)
			assert.Nil(t, err)

			var meetingLinkTransformer KeepMeetingLink
			event, err = meetingLinkTransformer.Transform(source, event)

			assert.Nil(t, err)

			expectedEvent := models.Event{
				Title:       expectedTitle,
				Description: tc.expectedDescription,
			}
			assert.Equal(t, expectedEvent, event)
		})
	}
}
