package sync

import (
	"context"
	"time"

	"github.com/inovex/CalendarSync/internal/models"
)

// RecurrencePattern is an RFC5545 conform recurrence pattern
type RecurrencePattern string

// NamedComponent is anything which can be named :).
type NamedComponent interface {
	// Name returns the name of the NamedComponent. This is used in debugging or error messages mostly.
	Name() string
}

// Source is a NamedComponent which describes the interface of a system (API) which acts as a source of events (aka calendar).
// A source can only ever be read from. No modifications can be performed on event-sources.
type Source interface {
	NamedComponent
	// EventsInTimeframe return all events in a certain timeframe
	EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error)
	GetCalendarID() string
}

// Sink describes a NamedComponent allows write-access to events.
// A sink has the capability to write events and is most likely a different calendar than the source.
type Sink interface {
	NamedComponent
	EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error)
	// CreateEvent creates the given event in the external calendar
	CreateEvent(ctx context.Context, e models.Event) error
	// UpdateEvent will update the given event
	UpdateEvent(ctx context.Context, e models.Event) error
	// DeleteEvent deletes the given Event in the external calendar
	DeleteEvent(ctx context.Context, e models.Event) error
	GetCalendarID() string
}
