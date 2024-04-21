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
	// If the user enters invalid hours, such as numbers higher than 24 or lower than 0, all events will be synchronized up to that point in time. For example, if the start hour is -1 and the end hour is 17, all events until 17 o'clock will be synchronized.
	if event.StartTime.Hour() >= a.HourStart && event.StartTime.Hour() <= a.HourEnd {
		return true
	}
	return false
}
