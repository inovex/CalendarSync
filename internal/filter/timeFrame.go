package filter

import (
	"github.com/inovex/CalendarSync/internal/models"
)

type TimeFrameEvents struct {
	HourStart int
	HourEnd   int
}

func (a TimeFrameEvents) Name() string {
	return "TimeFrame"
}

func (a TimeFrameEvents) Filter(event models.Event) bool {
	// if it is an all-day event, it should not be filtered here,
	// the AllDayEvents filter should be used instead
	if event.AllDay {
		return true
	}

	// if start time is inside the timeframe
	// example: event from 10-12, timeframe is 8-18
	// starttime and endtime are inside the timeframe
	if event.StartTime.Hour() >= a.HourStart && event.StartTime.Hour() <= a.HourEnd {
		return true
	}

	// if the endtime is inside the timeframe
	// example: event from 7-9, timeframe is 8-18
	// then the endtime of the event is inside the timeframe and therefore should be kept
	if event.EndTime.Hour() <= a.HourEnd && event.EndTime.Hour() >= a.HourStart {
		return true
	}

	// if the starttime is inside the timeframe
	// example: event from 17-19 timeframe is 8-18
	// then the starttime of the event is inside the timeframe and therefore should be kept
	if event.StartTime.Hour() >= a.HourStart && event.StartTime.Hour() <= a.HourEnd {
		return true
	}

	return false
}
