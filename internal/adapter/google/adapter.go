package google

import (

	//	"encoding/json"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/browser"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"

	"golang.org/x/oauth2/google"

	log "github.com/sirupsen/logrus"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/auth"
	"golang.org/x/oauth2"
	"google.golang.org/api/calendar/v3"
)

type GoogleCalendarClient interface {
	ListEvents(ctx context.Context, starttime time.Time, enddtime time.Time) ([]models.Event, error)
	CreateEvent(ctx context.Context, event models.Event) error
	UpdateEvent(ctx context.Context, event models.Event) error
	DeleteEvent(ctx context.Context, event models.Event) error
	GetSourceID() string
	InitGoogleCalendarClient(calId string) error
}

// TODO: add filter mechanism to clients for selective sync

// CalendarAPI is our Google Calendar client wrapper which adapts the base api to the needs of CalendarSync.
type CalendarAPI struct {
	gcalClient     GoogleCalendarClient
	logger         *log.Entry
	pageMaxResults int64
	calendarID     string

	authenticated bool
	oAuthUrl      string
	oAuthToken    *oauth2.Token
	oAuthHandler  auth.OAuthHandler

	storage auth.Storage
}

func (c *CalendarAPI) SetupOauth2(credentials auth.Credentials, storage auth.Storage) error {
	// Google Adapter does not need the tenantId
	switch {
	case credentials.Client.Id == "":
		return fmt.Errorf("%s adapter oAuth2 'clientID' cannot be empty", c.Name())
	case credentials.Client.Secret == "":
		return fmt.Errorf("oAuth2 adapter (%s) 'clientSecret' cannot be empty", c.Name())
	case credentials.CalendarId == "":
		return fmt.Errorf("oAuth2 adapter (%s) 'calendar' cannot be empty", c.Name())
	}

	oauthListener, err := auth.NewOAuthHandler(oauth2.Config{
		ClientID:     credentials.Client.Id,
		ClientSecret: credentials.Client.Secret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{calendar.CalendarReadonlyScope, calendar.CalendarEventsScope},
	})
	if err != nil {
		return err
	}

	c.oAuthHandler = oauthListener
	c.calendarID = credentials.CalendarId
	c.storage = storage

	storedAuth, err := c.storage.ReadCalendarAuth(credentials.CalendarId)
	if err != nil {
		return err
	}

	if storedAuth != nil {
		expiry, err := time.Parse(time.RFC3339, storedAuth.OAuth2.Expiry)
		if err != nil {
			return err
		}

		c.oAuthToken = &oauth2.Token{
			AccessToken:  storedAuth.OAuth2.AccessToken,
			RefreshToken: storedAuth.OAuth2.RefreshToken,
			Expiry:       expiry,
			TokenType:    storedAuth.OAuth2.TokenType,
		}

		c.authenticated = true
		c.logger.Infof("using stored credentials")
	}

	return nil
}

// Initialize implements the Configurable interface and allows the adapter to be dynamically configured.
// The given config is presumably unknown and is validated and loaded in order to construct a valid
// CalendarAPI struct.
// If anything fails, an error is returned and the CalendarAPI should be considered non-functional.
func (c *CalendarAPI) Initialize(ctx context.Context, config map[string]interface{}) error {
	if !c.authenticated {
		c.oAuthUrl = c.oAuthHandler.Configuration().AuthCodeURL("state", oauth2.AccessTypeOffline)
		c.logger.WithFields(log.Fields{}).Infof("opening browser window for authentication of %s\n", c.Name())
		err := browser.OpenURL(c.oAuthUrl)
		if err != nil {
			c.logger.WithFields(log.Fields{}).Infof("browser did not open, please authenticate adapter %s:\n\n %s\n\n\n", c.Name(), c.oAuthUrl)
		}

		if err := c.oAuthHandler.Listen(ctx); err != nil {
			return err
		}

		c.oAuthToken = c.oAuthHandler.Token()
		_, err = c.storage.WriteCalendarAuth(auth.CalendarAuth{
			CalendarID: c.calendarID,
			OAuth2: auth.OAuth2Object{
				AccessToken:  c.oAuthToken.AccessToken,
				RefreshToken: c.oAuthToken.RefreshToken,
				Expiry:       c.oAuthToken.Expiry.Format(time.RFC3339),
				TokenType:    c.oAuthToken.TokenType,
			},
		})
		if err != nil {
			return err
		}

	} else {
		c.logger.Debugln("adapter is already authenticated, loading access token")
	}

	c.pageMaxResults = defaultPageMaxResults
	// TODO: this does not seem right
	c.gcalClient = &GCalClient{oauthClient: c.oAuthHandler.Configuration().Client(ctx, c.oAuthToken)}
	err := c.gcalClient.InitGoogleCalendarClient(c.calendarID)
	if err != nil {
		return err
	}

	// Check if the token is expired:
	// {
	// "error": "invalid_grant",
	// "error_description": "Token has been expired or revoked."
	// }
	_, err = c.gcalClient.ListEvents(ctx, time.Now().Add(-3*time.Hour), time.Now().Add(3*time.Hour))
	if err != nil {
		if strings.Contains(err.Error(), "Token has been expired") {
			c.logger.Infof("the refresh token expired, initiating reauthentication...")
			err := c.storage.RemoveCalendarAuth(c.calendarID)
			if err != nil {
				return fmt.Errorf("failed to remove authentication for calendar %s: %w", c.calendarID, err)
			}
			c.authenticated = false
			err = c.Initialize(ctx, config)
			if err != nil {
				return fmt.Errorf("couldn't reinitialize calendar after expired refresh token: %w", err)
			}
			return nil
		}
		return err
	}
	return nil
}

// EventsInTimeframe returns all events in a Google calendar within the given start and end times.
func (c *CalendarAPI) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	events, err := c.gcalClient.ListEvents(ctx, start, end)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"adapter": c.Name(),
	}).Printf("loaded %d events between %s and %s.", len(events), start.Format(time.RFC1123), end.Format(time.RFC1123))

	return events, nil
}

// CreateEvent inserts a new event in the configured Google Calendar based on a given sync.Event.
func (c *CalendarAPI) CreateEvent(ctx context.Context, e models.Event) error {
	err := c.gcalClient.CreateEvent(ctx, e)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"adapter": c.Name(),
	}).Printf("Event %s at %s created", e.ShortTitle(), e.StartTime.String())

	return nil
}

// UpdateEvent updates an event in the calendar with the given sync.Event.
// The event which is going to be updated must have the same ID as the given sync.Event.
// Custom metadata inserted by CalendarSync will be added to the EventExtendedProperties.
func (c *CalendarAPI) UpdateEvent(ctx context.Context, e models.Event) error {
	err := c.gcalClient.UpdateEvent(ctx, e)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"adapter": c.Name(),
	}).Printf("Event %s at %s updated", e.ShortTitle(), e.StartTime.String())

	return nil
}

// DeleteEvent removes the given event from the calendar.
func (c *CalendarAPI) DeleteEvent(ctx context.Context, e models.Event) error {
	err := c.gcalClient.DeleteEvent(ctx, e)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"adapter": c.Name(),
	}).Printf("Event %s at %s deleted", e.ShortTitle(), e.StartTime.String())

	return nil
}

// SetLogger implements the LogSetter interface and allows injection of a custom logrus.Entry into the client.
func (c *CalendarAPI) SetLogger(logger *log.Entry) {
	c.logger = logger
}

// Name implements the NamedComponent interface and provides a very fancy name.
func (c *CalendarAPI) Name() string {
	return "Google Calendar"
}

// GetSourceID calculates a unique SourceID for this adapter based on the current calendar.
// This is used to distinguish between adapters in order to not overwrite or delete events
// which are maintained by different adapters.
// A simple use-case for this is if you have multiple google calendars as source adapters configured.
func (c *CalendarAPI) GetSourceID() string {
	return c.gcalClient.GetSourceID()
}
