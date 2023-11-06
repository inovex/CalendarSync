package filter

import (
	"github.com/inovex/CalendarSync/internal/models"
)

// Need to declare another interface similar to the sync.Filter interface here, otherwise we would land in an import loop
type MockFilter interface {
	Filter(models.Event) bool
}

// FilterEvents takes an array of events and a filter and executes the .Filter Method on each of the sourceEvents
// Not exluded events get returned in the filteredEvents
func FilterEvents(sourceEvents []models.Event, filter MockFilter) (filteredEvents []models.Event) {
	for _, event := range sourceEvents {
		if filter.Filter(event) {
			filteredEvents = append(filteredEvents, event)
		}
	}
	return filteredEvents
}
