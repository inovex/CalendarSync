package filter_test

import (
	"testing"
	"time"

	"github.com/inovex/CalendarSync/internal/filter"
	"github.com/inovex/CalendarSync/internal/models"
)

const timeFormat string = "2006-01-02T15:04"

// Events which match the start and end hour should be kept
func TestTimeFrameEventsFilter(t *testing.T) {
	t1, err := time.Parse(timeFormat, "2024-01-01T13:00")
	if err != nil {
		t.Error(err)
	}

	t2, err := time.Parse(timeFormat, "2024-01-01T18:00")
	if err != nil {
		t.Error(err)
	}

	t3, err := time.Parse(timeFormat, "2024-01-01T07:00")
	if err != nil {
		t.Error(err)
	}

	sourceEvents := []models.Event{
		// Should be kept, as Start AND Endtime is inside the timeframe
		{
			ICalUID:     "testId",
			ID:          "testUid",
			Title:       "test",
			Description: "bar",
			AllDay:      true,
			StartTime:   t1,
			EndTime:     t1.Add(time.Hour * 1),
		},
		// Should be filtered, as the Start and EndTime is out of range
		{
			ICalUID:     "testId2",
			ID:          "testUid2",
			Title:       "Test 2",
			Description: "bar",
			AllDay:      false,
			StartTime:   t2,
			EndTime:     t2.Add(time.Hour * 1),
		},
		// Should be kept
		{
			ICalUID:     "testId3",
			ID:          "testUid2",
			Title:       "foo",
			Description: "bar",
			StartTime:   t1,
			EndTime:     t1.Add(time.Hour * 1),
		},
		// Should be kept, as the end time is part of the timeframe
		{
			ICalUID:     "testId4",
			ID:          "testUid3",
			Title:       "foo",
			Description: "bar",
			StartTime:   t3,
			EndTime:     t3.Add(time.Hour * 1),
		},
		// Should be kept, as the start time is part of the timeframe
		{
			ICalUID:     "testId5",
			ID:          "testUid4",
			Title:       "foo",
			Description: "bar",
			StartTime:   t1,
			EndTime:     t1.Add(time.Hour * 7),
		},
	}

	expectedSinkEvents := []models.Event{sourceEvents[0], sourceEvents[2], sourceEvents[3], sourceEvents[4]}

	eventFilter := filter.TimeFrameEvents{
		// Events outside 8 am and 5 pm will be filtered.
		HourStart: 8,
		HourEnd:   17,
	}
	checkEventFilter(t, eventFilter, sourceEvents, expectedSinkEvents)
}
