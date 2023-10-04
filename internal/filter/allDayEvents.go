package filter

import (
	"github.com/inovex/CalendarSync/internal/models"
)

type AllDayEvents struct {
}

func (a AllDayEvents) Name() string {
	return "AllDayEvents"
}

func (a AllDayEvents) Filter(event models.Event) bool {
	return !event.AllDay
}
