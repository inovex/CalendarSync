package transformation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/inovex/CalendarSync/internal/models"
)

// verify keep attendees
func TestKeepAttendeesWithAnonymousEmail(t *testing.T) {
	source := models.Event{
		ICalUID:     "testId",
		ID:          "testUid",
		Title:       "foo",
		Description: "bar",
		Attendees: []models.Attendee{
			{
				DisplayName: "Foo",
				Email:       "foo@example.com",
			},
			{
				DisplayName: "Bar",
				Email:       "bar@example.com",
			},
		},
	}
	sink := models.NewSyncEvent(source)

	sut := KeepAttendees{
		UseEmailAsDisplayName: false,
	}

	event, err := sut.Transform(source, sink)

	assert.Nil(t, err)
	expectedEvent := models.Event{
		ICalUID: "testId",
		ID:      "testUid",
		Title:   "CalendarSync Event",
		Attendees: []models.Attendee{
			{
				DisplayName: "Foo",
				Email:       "foo_example.com@localhost",
			},
			{
				DisplayName: "Bar",
				Email:       "bar_example.com@localhost",
			},
		},
	}

	assert.Equal(t, expectedEvent, event)
}

// verify keep attendees with email as display name
func TestKeepAttendeesWithEmailAsDisplayName(t *testing.T) {
	source := models.Event{
		ICalUID:     "testId",
		ID:          "testUid",
		Title:       "foo",
		Description: "bar",
		Attendees: []models.Attendee{
			{
				DisplayName: "Foo",
				Email:       "foo@example.com",
			},
			{
				DisplayName: "Bar",
				Email:       "bar@example.com",
			},
		},
	}
	sink := models.NewSyncEvent(source)

	transformer := KeepAttendees{
		UseEmailAsDisplayName: true,
	}

	event, err := transformer.Transform(source, sink)

	assert.Nil(t, err)
	expectedEvent := models.Event{
		ICalUID: "testId",
		ID:      "testUid",
		Title:   "CalendarSync Event",
		Attendees: []models.Attendee{
			{
				DisplayName: "foo@example.com",
				Email:       "foo_example.com@localhost",
			},
			{
				DisplayName: "bar@example.com",
				Email:       "bar_example.com@localhost",
			},
		},
	}
	assert.Equal(t, expectedEvent, event)
}
