package filter_test

import (
	"testing"

	"github.com/inovex/CalendarSync/internal/filter"
	"github.com/inovex/CalendarSync/internal/models"
)

// Declined Events should be filtered
func TestDeclinedEventsFilter(t *testing.T) {
	sourceEvents := []models.Event{
		{
			ICalUID:     "testId",
			ID:          "testUid",
			Title:       "test",
			Description: "bar",
			Accepted:    false,
		},
		{
			ICalUID:     "testId2",
			ID:          "testUid2",
			Title:       "Test 2",
			Description: "bar",
			Accepted:    true,
		},
		{
			ICalUID:     "testId3",
			ID:          "testUid3",
			Title:       "foo",
			Description: "bar",
			Accepted:    true,
		},
	}

	expectedSinkEvents := []models.Event{sourceEvents[1], sourceEvents[2]}

	eventFilter := filter.DeclinedEvents{}
	checkEventFilter(t, eventFilter, sourceEvents, expectedSinkEvents)
}
