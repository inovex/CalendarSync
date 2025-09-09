package apple

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"

	"github.com/inovex/CalendarSync/internal/adapter/port"
	"github.com/inovex/CalendarSync/internal/auth"
	"github.com/inovex/CalendarSync/internal/models"
)

const (
	baseUrl    = "https://caldav.icloud.com"
	timeFormat = "20060102T150405Z"
)

type AppleCalendarClient interface {
	ListEvents(ctx context.Context, starttime time.Time, endtime time.Time) ([]models.Event, error)
	CreateEvent(ctx context.Context, event models.Event) error
	UpdateEvent(ctx context.Context, event models.Event) error
	DeleteEvent(ctx context.Context, event models.Event) error
	ResolveCalendarID(ctx context.Context, calendarID string) (string, string, error)
	DiscoverCalendars(ctx context.Context) ([]string, error)
	GetCalendarHash() string
}

type CalendarAPI struct {
	appleClient AppleCalendarClient
	calendarID  string

	username      string
	appPassword   string
	authenticated bool

	logger *log.Logger

	storage auth.Storage
}

// Assert that the expected interfaces are implemented
var _ port.Configurable = &CalendarAPI{}
var _ port.LogSetter = &CalendarAPI{}

func (c *CalendarAPI) SetupAuth(ctx context.Context, credentials auth.Credentials, storage auth.Storage, bindPort uint) error {
	// Reuse OAuth fields for basic auth:
	// clientId = Apple ID
	// clientSecret = App-specific password
	// tenantId = unused (empty)

	switch {
	case credentials.Client.Id == "":
		return fmt.Errorf("%s adapter 'clientId' (Apple ID) cannot be empty", c.Name())
	case credentials.Client.Secret == "":
		return fmt.Errorf("%s adapter 'clientSecret' (app-specific password) cannot be empty", c.Name())
	case credentials.CalendarId == "":
		return fmt.Errorf("%s adapter 'calendar' cannot be empty", c.Name())
	}

	c.calendarID = credentials.CalendarId
	c.username = credentials.Client.Id
	c.appPassword = credentials.Client.Secret
	c.storage = storage
	c.authenticated = true
	return nil
}

// Stub for interface compliance
func (c *CalendarAPI) SetupOauth2(ctx context.Context, credentials auth.Credentials, storage auth.Storage, bindPort uint) error {
	return c.SetupAuth(ctx, credentials, storage, bindPort)
}

func (c *CalendarAPI) SetupBasicAuth(ctx context.Context, credentials auth.Credentials, storage auth.Storage) error {
	switch {
	case credentials.Client.Id == "": // Using Client.Id as username for Apple ID
		return fmt.Errorf("%s adapter 'username' (Apple ID) cannot be empty", c.Name())
	case credentials.Client.Secret == "": // Using Client.Secret as app-specific password
		return fmt.Errorf("%s adapter 'appPassword' cannot be empty", c.Name())
	case credentials.CalendarId == "":
		return fmt.Errorf("%s adapter 'calendar' cannot be empty", c.Name())
	}

	c.calendarID = credentials.CalendarId
	c.username = credentials.Client.Id
	c.appPassword = credentials.Client.Secret
	c.storage = storage

	// For Apple CalDAV, we don't need OAuth2, just basic auth with app-specific password
	c.authenticated = true
	c.logger.Debug("Apple adapter configured with basic auth")

	return nil
}

func (c *CalendarAPI) Initialize(ctx context.Context, openBrowser bool, config map[string]interface{}) error {
	if !c.authenticated {
		return fmt.Errorf("Apple adapter not properly authenticated")
	}

	c.appleClient = &ACalClient{
		Username:    c.username,
		AppPassword: c.appPassword,
		CalendarID:  c.calendarID,
	}

	// Verify calendar resolution and discovery
	calendars, err := c.appleClient.DiscoverCalendars(ctx)
	if err != nil {
		log.Errorf("Calendar discovery failed: %v", err)
	} else {
		log.Infof("Discovered calendars: %v", calendars)

		principalID, resolvedID, err := c.appleClient.(*ACalClient).getResolvedCalendarInfo(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve calendar '%s': %w", c.calendarID, err)
		}
		log.Infof("Successfully resolved calendar '%s' to %s/%s", c.calendarID, principalID, resolvedID)
	}

	c.logger.Debug("Apple CalDAV client initialized")
	return nil
}

func (c *CalendarAPI) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	events, err := c.appleClient.ListEvents(ctx, start, end)
	if err != nil {
		return nil, err
	}

	c.logger.Infof("loaded %d events between %s and %s.", len(events), start.Format(time.DateOnly), end.Format(time.DateOnly))

	return events, nil
}

func (c *CalendarAPI) CreateEvent(ctx context.Context, e models.Event) error {
	err := c.appleClient.CreateEvent(ctx, e)
	if err != nil {
		return err
	}

	c.logger.Info("Event created", "title", e.ShortTitle(), "time", e.StartTime.String())

	return nil
}

func (c *CalendarAPI) UpdateEvent(ctx context.Context, e models.Event) error {
	err := c.appleClient.UpdateEvent(ctx, e)
	if err != nil {
		return err
	}

	c.logger.Info("Event updated", "title", e.ShortTitle(), "time", e.StartTime.String())

	return nil
}

func (c *CalendarAPI) DeleteEvent(ctx context.Context, e models.Event) error {
	err := c.appleClient.DeleteEvent(ctx, e)
	if err != nil {
		return err
	}

	c.logger.Info("Event deleted", "title", e.ShortTitle(), "time", e.StartTime.String())

	return nil
}

func (c *CalendarAPI) GetCalendarHash() string {
	return c.appleClient.GetCalendarHash()
}

func (c *CalendarAPI) Name() string {
	return "Apple CalDAV"
}

func (c *CalendarAPI) SetLogger(logger *log.Logger) {
	c.logger = logger
}
