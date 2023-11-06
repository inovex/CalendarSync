package filter_test

import (
	"testing"

	"github.com/inovex/CalendarSync/internal/filter"
	"github.com/inovex/CalendarSync/internal/models"
	"github.com/stretchr/testify/assert"
)

var sourceEvents = []models.Event{
	{
		ICalUID:     "testId",
		ID:          "testUid",
		Title:       "test",
		Description: "bar",
	},
	{
		ICalUID:     "testId2",
		ID:          "testUid2",
		Title:       "Test 2",
		Description: "bar",
	},
	{
		ICalUID:     "testId3",
		ID:          "testUid2",
		Title:       "foo",
		Description: "bar",
	},
}

// Some events should be filtered
func TestRegexTitleFilter(t *testing.T) {

	expectedSinkEvents := []models.Event{sourceEvents[1], sourceEvents[2]}

	eventFilter := filter.RegexTitle{
		ExludeRegexp: ".*test",
	}

	filteredEvents := FilterEvents(sourceEvents, eventFilter)

	assert.Equal(t, expectedSinkEvents, filteredEvents)
}

// All Events should be there
func TestRegexTitleFilterEmptyRegex(t *testing.T) {
	expectedSinkEvents := sourceEvents

	eventFilter := filter.RegexTitle{
		ExludeRegexp: "",
	}

	filteredEvents := FilterEvents(sourceEvents, eventFilter)

	assert.Equal(t, expectedSinkEvents, filteredEvents)
}
