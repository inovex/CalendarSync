package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/inovex/CalendarSync/internal/auth"
	"github.com/inovex/CalendarSync/internal/models"

	"github.com/charmbracelet/log"

	"github.com/inovex/CalendarSync/internal/adapter/google"
	"github.com/inovex/CalendarSync/internal/adapter/outlook"
	"github.com/inovex/CalendarSync/internal/adapter/port"
	"github.com/inovex/CalendarSync/internal/sync"
)

type SinkAdapter struct {
	client     sync.Sink
	calendarID string
	typ        Type
	logger     *log.Logger
}

// SinkClientFactory is a convenience factory. It is needed to retrieve new - default - client implementations.
func SinkClientFactory(typ Type) (sync.Sink, error) {
	switch typ {
	case GoogleCalendarType:
		return new(google.CalendarAPI), nil
	case OutlookHttpCalendarType:
		return new(outlook.HttpCalendarAPI), nil
	case OutlookTokenCalendarType:
		return new(outlook.TokenCalendarAPI), nil
	default:
		return nil, fmt.Errorf("unknown sink adapter client type %s", typ)
	}
}

func NewSinkAdapterFromConfig(ctx context.Context, bindPort uint, openBrowser bool, config ConfigReader, storage auth.Storage, logger *log.Logger) (*SinkAdapter, error) {
	client, err := SinkClientFactory(Type(config.Adapter().Type))
	if err != nil {
		return nil, err
	}

	if c, ok := client.(port.LogSetter); ok {
		c.SetLogger(logger)
	}

	if c, ok := client.(port.CalendarIDSetter); ok {
		if err := c.SetCalendarID(config.Adapter().Calendar); err != nil {
			return nil, err
		}
	}

	if c, ok := client.(port.StorageSetter); ok {
		c.SetStorage(storage)
	}

	if c, ok := client.(port.OAuth2Adapter); ok {
		if err := c.SetupOauth2(ctx,
			auth.Credentials{
				Client: auth.Client{
					Id:     config.Adapter().OAuth.ClientID,
					Secret: config.Adapter().OAuth.ClientKey,
				},
				Tenant: auth.Tenant{
					Id: config.Adapter().OAuth.TenantID,
				},
			},
			storage,
			bindPort,
		); err != nil {
			return nil, err
		}
	}

	// configure adapter client if possible
	if c, ok := client.(port.Configurable); ok {
		if err := c.Initialize(ctx, openBrowser, config.Adapter().Config); err != nil {
			return nil, fmt.Errorf("unable to Initialize adapter %s: %w", config.Adapter().Type, err)
		}
	}

	return &SinkAdapter{
		client:     client,
		calendarID: config.Adapter().Calendar,
		typ:        Type(config.Adapter().Type),
		logger:     logger,
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

func (a SinkAdapter) GetCalendarHash() string {
	return a.client.GetCalendarHash()
}
