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
	if event.StartTime.Hour() >= a.HourStart && event.StartTime.Hour() <= a.HourEnd {
		return true
	}
	return false
}
