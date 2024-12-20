package adapter

import (
	"github.com/inovex/CalendarSync/internal/config"
)

type Type string

const (
	GoogleCalendarType           Type = "google"
	ZepCalendarType              Type = "zep"
	OutlookHttpCalendarType      Type = "outlook_http"
	OutlookPublishedCalendarType Type = "outlook_published"
)

// ConfigReader provides an interface for adapters to load their own configuration map.
// It's the adapter's responsibility to validate that the map is valid.
type ConfigReader interface {
	Adapter() config.Adapter
}
