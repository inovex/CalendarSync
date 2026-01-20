package transformation

import (
	"testing"

	"github.com/inovex/CalendarSync/internal/models"
	"github.com/stretchr/testify/assert"
)

// verify keep attendees
func TestAddOriginalLink_Transform(t *testing.T) {
	source := models.Event{
		ICalUID:     "testId",
		ID:          "testUid",
		Title:       "foo",
		Description: "bar",
		HTMLLink:    "https://foo.com/bar?calendarId=testId",
	}
	sink := models.NewSyncEvent(source)

	addOriginalLink := AddOriginalLink{}

	event, err := addOriginalLink.Transform(source, sink)

	assert.Nil(t, err)
	expectedEvent := models.Event{
		ICalUID:     "testId",
		ID:          "testUid",
		Title:       "CalendarSync Event",
		Description: "original event link: https://foo.com/bar?calendarId=testId\n\n############\n",
	}

	assert.Equal(t, expectedEvent, event)
}

func TestAddOriginalLink_Transform_WithKeepDescription(t *testing.T) {
	source := models.Event{
		ICalUID:     "testId",
		ID:          "testUid",
		Title:       "foo",
		Description: "bar",
		HTMLLink:    "https://foo.com/bar?calendarId=testId",
	}
	sink := models.NewSyncEvent(source)

	keepDescription := KeepDescription{}
	addOriginalLink := AddOriginalLink{}

	sink, _ = keepDescription.Transform(source, sink)
	event, err := addOriginalLink.Transform(source, sink)

	assert.Nil(t, err)
	expectedEvent := models.Event{
		ICalUID:     "testId",
		ID:          "testUid",
		Title:       "CalendarSync Event",
		Description: "original event link: https://foo.com/bar?calendarId=testId\n\n############\nbar",
	}

	assert.Equal(t, expectedEvent, event)
}

func TestAddOriginalLink_Transform_WithKeepDescriptionAndKeepMeetingLink(t *testing.T) {
	source := models.Event{
		ICalUID:     "testId",
		ID:          "testUid",
		Title:       "foo",
		Description: "bar",
		MeetingLink: "https://meetinglink.de",
		HTMLLink:    "https://foo.com/bar?calendarId=testId",
	}
	sink := models.NewSyncEvent(source)

	keepDescription := KeepDescription{}
	keepMeetingLink := KeepMeetingLink{}
	addOriginalLink := AddOriginalLink{}

	sink, _ = keepDescription.Transform(source, sink)
	sink, _ = keepMeetingLink.Transform(source, sink)
	event, err := addOriginalLink.Transform(source, sink)

	assert.Nil(t, err)
	expectedEvent := models.Event{
		ICalUID: "testId",
		ID:      "testUid",
		Title:   "CalendarSync Event",
		Description: "original event link: https://foo.com/bar?calendarId=testId\n\n############\n" +
			"original meeting link: https://meetinglink.de\n\n############\nbar",
	}

	assert.Equal(t, expectedEvent, event)
}
