package filter_test

import (
	"github.com/inovex/CalendarSync/internal/models"
	"github.com/inovex/CalendarSync/internal/sync"
)

// FilterEvents takes an array of events and a filter and executes the .Filter Method on each of the sourceEvents
// Not exluded events get returned in the filteredEvents
func FilterEvents(sourceEvents []models.Event, filter sync.Filter) (filteredEvents []models.Event) {
	for _, event := range sourceEvents {
		if filter.Filter(event) {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return filteredEvents
}
