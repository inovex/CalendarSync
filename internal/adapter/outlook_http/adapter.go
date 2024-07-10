package outlook_http

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"

	"github.com/inovex/CalendarSync/internal/adapter/port"
	"github.com/inovex/CalendarSync/internal/auth"
	"github.com/inovex/CalendarSync/internal/models"
)

const (
	baseUrl    = "https://graph.microsoft.com/v1.0"
	timeFormat = "2006-01-02T15:04:05.0000000"
)

type OutlookCalendarClient interface {
	ListEvents(ctx context.Context, starttime time.Time, endtime time.Time) ([]models.Event, error)
	CreateEvent(ctx context.Context, event models.Event) error
	UpdateEvent(ctx context.Context, event models.Event) error
	DeleteEvent(ctx context.Context, event models.Event) error
	GetCalendarID() string
}

type CalendarAPI struct {
	outlookClient OutlookCalendarClient
	calendarID    string

	oAuthConfig   *oauth2.Config
	authenticated bool
	oAuthUrl      string
	oAuthToken    *oauth2.Token
	oAuthHandler  *auth.OAuthHandler

	logger *log.Logger

	storage auth.Storage
}

// Assert that the expected interfaces are implemented
var _ port.Configurable = &CalendarAPI{}
var _ port.LogSetter = &CalendarAPI{}
var _ port.OAuth2Adapter = &CalendarAPI{}

func (c *CalendarAPI) SetupOauth2(ctx context.Context, credentials auth.Credentials, storage auth.Storage, bindPort uint) error {
	// Outlook Adapter does not need the clientKey
	switch {
	case credentials.Client.Id == "":
		return fmt.Errorf("%s adapter oAuth2 'clientId' cannot be empty", c.Name())
	case credentials.Tenant.Id == "":
		return fmt.Errorf("%s adapter oAuth2 'tenantId' cannot be empty", c.Name())
	case credentials.CalendarId == "":
		return fmt.Errorf("%s adapter oAuth2 'calendar' cannot be empty", c.Name())
	}

	c.calendarID = credentials.CalendarId

	endpoint := oauth2.Endpoint{
		AuthURL:   fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", credentials.Tenant.Id),
		TokenURL:  fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", credentials.Tenant.Id),
		AuthStyle: oauth2.AuthStyleInParams,
	}

	oAuthConfig := oauth2.Config{
		ClientID: credentials.Client.Id,
		Endpoint: endpoint,
		Scopes:   []string{"Calendars.ReadWrite", "offline_access"}, // You need to request offline_access in order to retrieve a refresh token
	}

	oAuthListener, err := auth.NewOAuthHandler(oAuthConfig, bindPort)
	if err != nil {
		return err
	}

	c.oAuthHandler = oAuthListener
	c.storage = storage
	c.oAuthConfig = &oAuthConfig

	storedAuth, err := c.storage.ReadCalendarAuth(credentials.CalendarId)
	if err != nil {
		return err
	}
	if storedAuth != nil {
		expiry, err := time.Parse(time.RFC3339, storedAuth.OAuth2.Expiry)
		if err != nil {
			return err
		}

		now := time.Now()
		if now.After(expiry) {
			c.logger.Debugf("expiry time of stored token: %s", expiry.String())
			src := c.oAuthConfig.TokenSource(ctx, &oauth2.Token{
				AccessToken:  storedAuth.OAuth2.AccessToken,
				RefreshToken: storedAuth.OAuth2.RefreshToken,
				Expiry:       expiry,
				TokenType:    storedAuth.OAuth2.TokenType,
			})

			// refresh tokens as the access token is expired
			// there is a lot of confusion here and i still didn't understood all of the implications, but it works
			// for more info: https://github.com/golang/oauth2/issues/84
			newToken, err := src.Token()
			if err != nil {
				// most probably the refresh token is now also expired
				c.logger.Info("saved credentials expired, we need to reauthenticate..", "error", err)
				c.authenticated = false
				err := c.storage.RemoveCalendarAuth(c.calendarID)
				if err != nil {
					return fmt.Errorf("failed to remove authentication for calendar %s: %w", c.calendarID, err)
				}
				return nil
			}
			// give our CalendarAPI the new token
			c.oAuthToken = &oauth2.Token{
				AccessToken:  newToken.AccessToken,
				RefreshToken: newToken.RefreshToken,
				Expiry:       newToken.Expiry,
				TokenType:    newToken.TokenType,
			}

			c.authenticated = true
			c.logger.Debug("Refreshed oauth credentials using the refresh token")
			c.logger.Debugf("expiry time of new token: %s", newToken.Expiry.String())

			// save the updated token to disk for the next use
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

			c.logger.Debug("saved new token to disk")
			return nil
		}

		// if the token isn't expired, load from disk and use it
		c.oAuthToken = &oauth2.Token{
			AccessToken:  storedAuth.OAuth2.AccessToken,
			RefreshToken: storedAuth.OAuth2.RefreshToken,
			Expiry:       expiry,
			TokenType:    storedAuth.OAuth2.TokenType,
		}

		c.authenticated = true
		c.logger.Info("using stored credentials")
		c.logger.Debugf("expiry time of stored token: %s", expiry.String())
	}

	return nil
}

func (c *CalendarAPI) Initialize(ctx context.Context, openBrowser bool, config map[string]interface{}) error {
	if !c.authenticated {
		c.oAuthUrl = c.oAuthHandler.Configuration().AuthCodeURL("state", oauth2.AccessTypeOffline)

		if openBrowser {
			c.logger.Infof("opening browser window for authentication of %s\n", c.Name())
			err := browser.OpenURL(c.oAuthUrl)
			if err != nil {
				c.logger.Infof("browser did not open, please authenticate adapter %s:\n\n %s\n\n\n", c.Name(), c.oAuthUrl)
			}
		} else {
			c.logger.Infof("Please authenticate adapter %s:\n\n %s\n\n\n", c.Name(), c.oAuthUrl)
		}
		if err := c.oAuthHandler.Listen(ctx); err != nil {
			return err
		}

		c.oAuthToken = c.oAuthHandler.Token()
		_, err := c.storage.WriteCalendarAuth(auth.CalendarAuth{
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
		c.logger.Debug("adapter is already authenticated, loading access token")
	}

	client := c.oAuthConfig.Client(ctx, c.oAuthToken)

	c.outlookClient = &OutlookClient{Client: client, CalendarID: c.calendarID}
	return nil
}

func (c *CalendarAPI) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	events, err := c.outlookClient.ListEvents(ctx, start, end)
	if err != nil {
		return nil, err
	}

	c.logger.Infof("loaded %d events between %s and %s.", len(events), start.Format(time.RFC1123), end.Format(time.RFC1123))

	return events, nil
}

func (c *CalendarAPI) CreateEvent(ctx context.Context, e models.Event) error {
	err := c.outlookClient.CreateEvent(ctx, e)
	if err != nil {
		return err
	}

	c.logger.Info("Event created", "title", e.ShortTitle(), "time", e.StartTime.String())

	return nil
}

func (c *CalendarAPI) UpdateEvent(ctx context.Context, e models.Event) error {
	err := c.outlookClient.UpdateEvent(ctx, e)
	if err != nil {
		return err
	}

	c.logger.Info("Event updated", "title", e.ShortTitle(), "time", e.StartTime.String())

	return nil
}

func (c *CalendarAPI) DeleteEvent(ctx context.Context, e models.Event) error {
	err := c.outlookClient.DeleteEvent(ctx, e)
	if err != nil {
		return err
	}

	c.logger.Info("Event deleted", "title", e.ShortTitle(), "time", e.StartTime.String())

	return nil
}

func (c *CalendarAPI) GetCalendarID() string {
	return c.outlookClient.GetCalendarID()
}

func (c *CalendarAPI) Name() string {
	return "Outlook"
}

func (c *CalendarAPI) SetLogger(logger *log.Logger) {
	c.logger = logger
}
