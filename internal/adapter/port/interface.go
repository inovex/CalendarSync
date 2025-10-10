package port

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/inovex/CalendarSync/internal/auth"
)

// LogSetter can be implemented by a struct to allows injection of a logger instance
type LogSetter interface {
	SetLogger(logger *log.Logger)
}

// CalendarIDSetter can be implemented by a struct to allow setting the calendar ID
type CalendarIDSetter interface {
	SetCalendarID(calendarID string) error
}

// Configurable is an interface which defines how arbitrary configuration data can be passed
// to a struct which implements this interface. Clients should be configurable.
type Configurable interface {
	Initialize(ctx context.Context, openBrowser bool, config map[string]interface{}) error
}

type OAuth2Adapter interface {
	SetupOauth2(ctx context.Context, credentials auth.Credentials, storage auth.Storage, bindPort uint) error
}
