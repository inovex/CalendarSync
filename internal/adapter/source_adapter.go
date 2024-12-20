package adapter

import (
	"context"
	"fmt"
	"github.com/inovex/CalendarSync/internal/adapter/outlook_published"
	"time"

	"github.com/charmbracelet/log"
	"github.com/inovex/CalendarSync/internal/auth"
	"github.com/inovex/CalendarSync/internal/models"

	outlook "github.com/inovex/CalendarSync/internal/adapter/outlook_http"
	"github.com/inovex/CalendarSync/internal/adapter/port"

	"github.com/inovex/CalendarSync/internal/adapter/google"
	"github.com/inovex/CalendarSync/internal/adapter/zep"
	"github.com/inovex/CalendarSync/internal/sync"
)

// SourceClientFactory is a convenience factory. It is needed to retrieve new - default - client implementations.
func SourceClientFactory(typ Type) (sync.Source, error) {
	switch typ {
	case GoogleCalendarType:
		return new(google.CalendarAPI), nil
	case OutlookPublishedCalendarType:
		return new(outlook_published.CalendarAPI), nil
	case ZepCalendarType:
		return new(zep.CalendarAPI), nil
	case OutlookHttpCalendarType:
		return new(outlook.CalendarAPI), nil
	default:
		return nil, fmt.Errorf("unknown source adapter client type %s", typ)
	}
}

type SourceAdapter struct {
	client     sync.Source
	calendarID string
	typ        Type
	logger     *log.Logger
}

func NewSourceAdapterFromConfig(ctx context.Context, bindPort uint, openBrowser bool, config ConfigReader, storage auth.Storage, logger *log.Logger) (*SourceAdapter, error) {
	var client sync.Source
	client, err := SourceClientFactory(Type(config.Adapter().Type))
	if err != nil {
		return nil, err
	}

	if c, ok := client.(port.LogSetter); ok {
		c.SetLogger(logger)
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
				CalendarId: config.Adapter().Calendar,
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
			return nil, fmt.Errorf("unable to initialize adapter %s: %w", config.Adapter().Type, err)
		}
	}

	return &SourceAdapter{
		client:     client,
		calendarID: config.Adapter().Calendar,
		typ:        Type(config.Adapter().Type),
		logger:     logger,
	}, nil

}

func (a SourceAdapter) Name() string {
	return string(a.typ)
}

func (a SourceAdapter) CalendarID() string {
	return a.calendarID
}
func (a SourceAdapter) GetCalendarID() string {
	return a.client.GetCalendarID()
}

func (a SourceAdapter) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	return a.client.EventsInTimeframe(ctx, start, end)
}
