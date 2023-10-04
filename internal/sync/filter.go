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
	// Filter returns true to keep the event
	Filter(event models.Event) bool
}

// FilterEvent returns false if one of the filters rejects the event
func FilterEvent(event models.Event, filters ...Filter) (result bool) {
	for _, filter := range filters {
		// If the filter returns false (or: filters the event), then return false
		if !filter.Filter(event) {
			return false
		}
	}
	// otherwise keep the event
	return true
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
