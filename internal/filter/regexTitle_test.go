package filter_test

import (
	"testing"

	"github.com/inovex/CalendarSync/internal/filter"
	"github.com/inovex/CalendarSync/internal/models"
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
		Title:       "taste",
		Description: "bar",
	},
}

// Some events should be filtered
func TestRegexTitleFilterExclude(t *testing.T) {

	expectedSinkEvents := []models.Event{sourceEvents[1], sourceEvents[2]}

	eventFilter := filter.RegexTitle{
		ExcludeRegexp: ".*test",
	}
	checkEventFilter(t, eventFilter, sourceEvents, expectedSinkEvents)
}

// All Events should be there
func TestRegexTitleFilterEmptyRegex(t *testing.T) {
	expectedSinkEvents := sourceEvents

	eventFilter := filter.RegexTitle{
		ExcludeRegexp: "",
	}
	checkEventFilter(t, eventFilter, sourceEvents, expectedSinkEvents)
}

// Only the included events should be there
func TestRegexTitleFilterInclude(t *testing.T) {
	expectedSinkEvents := []models.Event{sourceEvents[0], sourceEvents[1]}
	eventFilter := filter.RegexTitle{
		ExcludeRegexp: ".*",
		IncludeRegexp: "[tT]est.*",
	}
	checkEventFilter(t, eventFilter, sourceEvents, expectedSinkEvents)
}

// the excluded events should be excluded but the included should be included again
func TestRegexTitleFilterExcludeInclude(t *testing.T) {
	expectedSinkEvents := []models.Event{sourceEvents[0]}
	eventFilter := filter.RegexTitle{
		ExcludeRegexp: "t.*",
		IncludeRegexp: "test.*",
	}
	checkEventFilter(t, eventFilter, sourceEvents, expectedSinkEvents)
}
