package filter

import (
	"testing"

	"github.com/inovex/CalendarSync/internal/models"
	"github.com/stretchr/testify/assert"
)

// Some events should be filtered
func TestRegexTitleFilter(t *testing.T) {
	sourceEvents := []models.Event{
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

	expectedSinkEvents := []models.Event{
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

	filter := RegexTitle{
		Regexp: ".*test",
	}

	filteredEvents := []models.Event{}

	for _, event := range sourceEvents {
		if filter.Filter(event) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	assert.Equal(t, expectedSinkEvents, filteredEvents)
}

// All Events should be there
func TestRegexTitleFilterEmptyRegex(t *testing.T) {
	sourceEvents := []models.Event{
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

	expectedSinkEvents := []models.Event{
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

	filter := RegexTitle{
		Regexp: "",
	}

	filteredEvents := []models.Event{}

	for _, event := range sourceEvents {
		if filter.Filter(event) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	assert.Equal(t, expectedSinkEvents, filteredEvents)
}
