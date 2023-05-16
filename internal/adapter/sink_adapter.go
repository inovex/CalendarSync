package adapter

import (
	"context"
	"fmt"
	"time"

	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/auth"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"

	log "github.com/sirupsen/logrus"

	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/adapter/google"
	outlook "gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/adapter/outlook_http"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/sync"
)

type SinkAdapter struct {
	client     sync.Sink
	calendarID string
	typ        Type
}

// SinkClientFactory is a convenience factory. It is needed to retrieve new - default - client implementations.
func SinkClientFactory(typ Type) (sync.Sink, error) {
	switch typ {
	case GoogleCalendarType:
		return new(google.CalendarAPI), nil
	case OutlookHttpCalendarType:
		return new(outlook.CalendarAPI), nil
	default:
		return nil, fmt.Errorf("unknown sink adapter client type %s", typ)
	}
}

func NewSinkAdapterFromConfig(ctx context.Context, config ConfigReader, storage auth.Storage) (*SinkAdapter, error) {
	client, err := SinkClientFactory(Type(config.Adapter().Type))
	if err != nil {
		return nil, err
	}

	logger := log.WithFields(log.Fields{
		"client":       config.Adapter().Type,
		"adapter_type": "sink",
		"calendar":     config.Adapter().Calendar,
	})

	if c, ok := client.(LogSetter); ok {
		c.SetLogger(logger)
	}

	if c, ok := client.(OAuth2Adapter); ok {
		if err := c.SetupOauth2(
			auth.Credentials{
				Client: auth.Client{
					Id:     config.Adapter().OAuth.ClientID,
					Secret: config.Adapter().OAuth.ClientKey,
				},
				Tenant: auth.Tenant{
					Id: config.Adapter().OAuth.TenantID,
				},
				CalendarId: config.Adapter().Calendar,
			},
			storage,
		); err != nil {
			return nil, err
		}
	}

	// configure adapter client if possible
	if c, ok := client.(Configurable); ok {
		if err := c.Initialize(ctx, config.Adapter().Config); err != nil {
			return nil, fmt.Errorf("unable to Initialize adapter %s: %w", config.Adapter().Type, err)
		}
	}

	return &SinkAdapter{
		client:     client,
		calendarID: config.Adapter().Calendar,
		typ:        Type(config.Adapter().Type),
	}, nil
}

func (a SinkAdapter) Name() string {
	return string(a.typ)
}

func (a SinkAdapter) CalendarID() string {
	return a.calendarID
}

func (a SinkAdapter) CreateEvent(ctx context.Context, e models.Event) error {
	err := a.client.CreateEvent(ctx, e)
	return err
}

func (a SinkAdapter) UpdateEvent(ctx context.Context, e models.Event) error {
	return a.client.UpdateEvent(ctx, e)
}

func (a SinkAdapter) DeleteEvent(ctx context.Context, e models.Event) error {
	return a.client.DeleteEvent(ctx, e)
}

func (a SinkAdapter) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	return a.client.EventsInTimeframe(ctx, start, end)
}
