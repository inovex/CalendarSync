package filter

import (
	"github.com/inovex/CalendarSync/internal/models"
)

type TimeFrameEvents struct {
	HourStart int64
	HourEnd   int64
}

func (a TimeFrameEvents) Name() string {
	return "TimeFrame"
}

func (a TimeFrameEvents) Filter(event models.Event) bool {
	if int64(event.StartTime.Hour()) >= a.HourStart && int64(event.StartTime.Hour()) <= a.HourEnd {
		return true
	}
	return false
}
