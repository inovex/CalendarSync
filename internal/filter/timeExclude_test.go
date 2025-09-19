package filter_test

import (
	"testing"
	"time"

	"github.com/inovex/CalendarSync/internal/filter"
	"github.com/inovex/CalendarSync/internal/models"
)

// Events which match the start and end hour should be kept
func TestTimeExcludeEventsFilter(t *testing.T) {
	const timeFormat string = "2006-01-02T15:04"

	t1, err := time.Parse(timeFormat, "2024-01-01T08:00")
	if err != nil {
		t.Error(err)
	}

	t2, err := time.Parse(timeFormat, "2024-01-01T12:15")
	if err != nil {
		t.Error(err)
	}

	t3, err := time.Parse(timeFormat, "2024-01-01T12:00")
	if err != nil {
		t.Error(err)
	}

	t4, err := time.Parse(timeFormat, "2024-01-01T13:00")
	if err != nil {
		t.Error(err)
	}

	sourceEvents := []models.Event{
		// Should be kept (all-day event)
		{
			ICalUID:     "testId",
			ID:          "testUid",
			Title:       "all-day",
			Description: "bar",
			AllDay:      true,
			StartTime:   t1,
			EndTime:     t1.Add(time.Hour * 1),
		},
		// Should be kept, as Start AND Endtime are outside the excluded times
		{
			ICalUID:     "testId",
			ID:          "testUid",
			Title:       "outside frame",
			Description: "bar",
			AllDay:      false,
			StartTime:   t1,
			EndTime:     t1.Add(time.Hour * 1),
		},
		// Should be filtered, as the Start and EndTime are inside the excluded times
		{
			ICalUID:     "testId2",
			ID:          "testUid2",
			Title:       "inside frame",
			Description: "bar",
			AllDay:      false,
			StartTime:   t2,
			EndTime:     t2.Add(time.Minute * 30),
		},
		// Should be filtered, as the end time is part of the timeframe
		{
			ICalUID:     "testId4",
			ID:          "inside frame 2",
			Title:       "foo",
			Description: "bar",
			StartTime:   t2,
			EndTime:     t2.Add(time.Minute * 45),
		},
		// Should be filtered, as start and end times are at exclusion borders
		{
			ICalUID:     "testId5",
			ID:          "testUid4",
			Title:       "inside frame 3",
			Description: "bar",
			StartTime:   t3,
			EndTime:     t3.Add(time.Hour * 1),
		},
		// Should be kept, as the start time is at the end of the exclusion border
		{
			ICalUID:     "testId5",
			ID:          "testUid4",
			Title:       "outside frame 2",
			Description: "bar",
			StartTime:   t4,
			EndTime:     t4.Add(time.Hour * 1),
		},
	}

	expectedSinkEvents := []models.Event{sourceEvents[0], sourceEvents[1], sourceEvents[5]}

	eventFilter := filter.TimeExcludeEvents{
		// Events from 12 to 13 should be filtered
		HourStart: 12,
		HourEnd:   13,
	}
	checkEventFilter(t, eventFilter, sourceEvents, expectedSinkEvents)
}
