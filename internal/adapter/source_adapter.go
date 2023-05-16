package adapter

import (
	"context"
	"fmt"
	"time"

	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/auth"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"

	outlook "gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/adapter/outlook_http"

	log "github.com/sirupsen/logrus"

	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/adapter/google"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/adapter/zep"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/sync"
)

// SourceClientFactory is a convenience factory. It is needed to retrieve new - default - client implementations.
func SourceClientFactory(typ Type) (sync.Source, error) {
	switch typ {
	case GoogleCalendarType:
		return new(google.CalendarAPI), nil
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
}

func NewSourceAdapterFromConfig(ctx context.Context, config ConfigReader, storage auth.Storage) (*SourceAdapter, error) {
	var client sync.Source
	client, err := SourceClientFactory(Type(config.Adapter().Type))
	if err != nil {
		return nil, err
	}

	logger := log.WithFields(log.Fields{
		"client":       config.Adapter().Type,
		"adapter_type": "source",
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
			storage); err != nil {
			return nil, err
		}
	}

	// configure adapter client if possible
	if c, ok := client.(Configurable); ok {
		if err := c.Initialize(ctx, config.Adapter().Config); err != nil {
			return nil, fmt.Errorf("unable to initialize adapter %s: %w", config.Adapter().Type, err)
		}
	}

	return &SourceAdapter{
		client:     client,
		calendarID: config.Adapter().Calendar,
		typ:        Type(config.Adapter().Type),
	}, nil

}

func (a SourceAdapter) Name() string {
	return string(a.typ)
}

func (a SourceAdapter) CalendarID() string {
	return a.calendarID
}
func (a SourceAdapter) GetSourceID() string {
	return a.client.GetSourceID()
}

func (a SourceAdapter) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	return a.client.EventsInTimeframe(ctx, start, end)
}
