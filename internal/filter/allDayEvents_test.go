package filter_test

import (
	"testing"

	"github.com/inovex/CalendarSync/internal/filter"
	"github.com/inovex/CalendarSync/internal/models"
)

// All Day Events should be filtered
func TestAllDayEventsFilter(t *testing.T) {
	sourceEvents := []models.Event{
		{
			ICalUID:     "testId",
			ID:          "testUid",
			Title:       "test",
			Description: "bar",
			AllDay:      true,
		},
		{
			ICalUID:     "testId2",
			ID:          "testUid2",
			Title:       "Test 2",
			Description: "bar",
			AllDay:      false,
		},
		{
			ICalUID:     "testId3",
			ID:          "testUid2",
			Title:       "foo",
			Description: "bar",
		},
	}

	expectedSinkEvents := []models.Event{sourceEvents[1], sourceEvents[2]}

	eventFilter := filter.AllDayEvents{}
	checkEventFilter(t, eventFilter, sourceEvents, expectedSinkEvents)
}
