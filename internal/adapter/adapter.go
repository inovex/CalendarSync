package adapter

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/inovex/CalendarSync/internal/auth"

	"github.com/inovex/CalendarSync/internal/config"
)

type Type string

const (
	GoogleCalendarType      Type = "google"
	ZepCalendarType         Type = "zep"
	OutlookCalendarType     Type = "outlook"
	OutlookHttpCalendarType Type = "outlook_http"
)

// LogSetter is a struct which implements this interface and thus allows injection of a logrus *log.Entry
type LogSetter interface {
	SetLogger(logger *log.Entry)
}

// Configurable is an interface which defines how arbitrary configuration data can be passed
// to a struct which implements this interface. Clients should be configurable.
type Configurable interface {
	Initialize(ctx context.Context, config map[string]interface{}) error
}

type OAuth2Adapter interface {
	SetupOauth2(credentials auth.Credentials, storage auth.Storage, bindPort uint) error
}

// ConfigReader provides an interface for adapters to load their own configuration map.
// It's the adapter's responsibility to validate that the map is valid.
type ConfigReader interface {
	Adapter() config.Adapter
}
