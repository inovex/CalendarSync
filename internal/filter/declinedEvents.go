package filter

import (
	"github.com/inovex/CalendarSync/internal/models"
)

type DeclinedEvents struct{}

func (d DeclinedEvents) Name() string {
	return "DeclinedEvents"
}

func (d DeclinedEvents) Filter(events []models.Event) (filteredEvents []models.Event) {
	for _, event := range events {
		if event.Accepted {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return filteredEvents
}
