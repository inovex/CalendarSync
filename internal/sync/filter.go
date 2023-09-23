package sync

import (
	"fmt"
	"strings"

	"github.com/inovex/CalendarSync/internal/config"
	"github.com/inovex/CalendarSync/internal/filter"
	"github.com/inovex/CalendarSync/internal/models"
)

type Filter interface {
	NamedComponent
	Filter(events []models.Event) []models.Event
}

func FilterEvents(events []models.Event, filters ...Filter) (filteredEvents []models.Event) {
	for _, filter := range filters {
		events = filter.Filter(events)
	}
	return events
}

var (
	filterConfigMapping = map[string]Filter{
		"DeclinedEvents": &filter.DeclinedEvents{},
		"AllDayEvents":   &filter.AllDayEvents{},
	}

	filterOrder = []string{
		"DeclinedEvents",
		"AllDayEvents",
	}
)

func FilterFactory(configuredFilters []config.Filter) (loadedFilters []Filter) {
	for _, configuredFilter := range configuredFilters {
		if _, nameExists := filterConfigMapping[configuredFilter.Name]; !nameExists {
			// todo: handle properly
			panic(fmt.Sprintf("unknown filter: %s", configuredFilter.Name))
		}
		// load the default Transformer for the configured name and initialize it based on the config
		filterDefault := filterConfigMapping[configuredFilter.Name]
		loadedFilters = append(loadedFilters, filterFromConfig(filterDefault, configuredFilter.Config))
	}

	var sortedAndLoadedFilter []Filter
	for _, name := range filterOrder {
		for _, v := range loadedFilters {
			if strings.EqualFold(name, v.Name()) {
				sortedAndLoadedFilter = append(sortedAndLoadedFilter, v)
			}
		}
	}

	return sortedAndLoadedFilter
}

func filterFromConfig(filter Filter, config config.CustomMap) Filter {
	autoConfigure(filter, config)
	return filter
}
