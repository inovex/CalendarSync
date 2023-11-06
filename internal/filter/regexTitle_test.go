package filter

import (
	"testing"

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

	filter := RegexTitle{
		ExludeRegexp: ".*test",
	}

	filteredEvents := FilterEvents(sourceEvents, filter)

	assert.Equal(t, expectedSinkEvents, filteredEvents)
}

// All Events should be there
func TestRegexTitleFilterEmptyRegex(t *testing.T) {
	expectedSinkEvents := sourceEvents

	filter := RegexTitle{
		ExludeRegexp: "",
	}

	filteredEvents := FilterEvents(sourceEvents, filter)

	assert.Equal(t, expectedSinkEvents, filteredEvents)
}
