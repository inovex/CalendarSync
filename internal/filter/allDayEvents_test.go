package filter

import (
	"testing"

	"github.com/inovex/CalendarSync/internal/models"
	"github.com/stretchr/testify/assert"
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

	filter := AllDayEvents{}

	filteredEvents := FilterEvents(sourceEvents, filter)

	assert.Equal(t, expectedSinkEvents, filteredEvents)
}
