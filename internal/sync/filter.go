package sync

import (
	"fmt"
	"reflect"
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

func autoConfigureFilter(filter Filter, config config.CustomMap) {
	ps := reflect.ValueOf(filter)
	s := ps.Elem()
	if s.Kind() == reflect.Struct {
		for key, value := range config {
			field := s.FieldByName(key)
			if field.IsValid() && field.CanSet() {
				switch field.Kind() {
				case reflect.Int,
					reflect.Int8,
					reflect.Int16,
					reflect.Int32,
					reflect.Int64:
					field.SetInt(value.(int64))
				case reflect.Bool:
					field.SetBool(value.(bool))
				case reflect.String:
					field.SetString(value.(string))
				default:
					panic(fmt.Sprintf("autoConfigure(): unknown kind '%s' for field '%s'", key, field.Kind().String()))
				}
			}
		}
	}
}

func filterFromConfig(filter Filter, config config.CustomMap) Filter {
	autoConfigureFilter(filter, config)
	return filter
}

func removeDuplicateInt(intSlice []int) []int {
	allKeys := make(map[int]bool)
	list := []int{}
	for _, item := range intSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
