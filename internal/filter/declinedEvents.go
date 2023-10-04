package filter

import (
	"github.com/inovex/CalendarSync/internal/models"
)

type DeclinedEvents struct{}

func (d DeclinedEvents) Name() string {
	return "DeclinedEvents"
}

func (d DeclinedEvents) Filter(event models.Event) bool {
	return event.Accepted
}
