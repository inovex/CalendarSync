package filter

import (
	"fmt"

	"github.com/inovex/CalendarSync/internal/models"
)

type AllDayEvents struct {
}

func (a AllDayEvents) Name() string {
	return "AllDayEvents"
}

func (a AllDayEvents) Filter(events []models.Event) (filteredEvents []models.Event) {
	for _, event := range events {
		if !event.AllDay {
			filteredEvents = append(filteredEvents, event)
		} else {
			fmt.Println("Filtered!")
		}
	}
	return filteredEvents
}
