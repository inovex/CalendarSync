package filter

import (
	"github.com/inovex/CalendarSync/internal/models"
)

type TimeExcludeEvents struct {
	HourStart int
	HourEnd   int
}

func (a TimeExcludeEvents) Name() string {
	return "TimeExclude"
}

func (a TimeExcludeEvents) Filter(event models.Event) bool {
	// if it is an all-day event, it should not be filtered here,
	// the AllDayEvents filter should be used instead
	if event.AllDay {
		return true
	}

	// if start time and end time are inside the timeframe: exclude
	// example: event from 12:15-12:45, timeframe is 12-13
	// starttime and endtime are inside the timeframe
	if event.StartTime.Hour() >= a.HourStart && event.EndTime.Hour() <= a.HourEnd {
		return false
	}

	// if the endtime is inside the timeframe, but the starttime is outside,
	// or if the starttime is inside the timeframe, but the endtime is outside,
	// or in any other case: keep the event
	return true
}
