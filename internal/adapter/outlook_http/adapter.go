package outlook_http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/browser"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

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
	GetSourceID() string
}

type CalendarAPI struct {
	logger        *log.Entry
	outlookClient OutlookCalendarClient
	calendarID    string

	oAuthConfig   *oauth2.Config
	authenticated bool
	oAuthUrl      string
	oAuthToken    *oauth2.Token
	oAuthHandler  *auth.OAuthHandler

	storage auth.Storage
}

func (c *CalendarAPI) SetupOauth2(credentials auth.Credentials, storage auth.Storage, bindPort uint) error {
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

	oAuthListener, err := auth.NewOAuthHandler(oauth2.Config{
		ClientID: credentials.Client.Id,
		Endpoint: endpoint,
		Scopes:   []string{"Calendars.ReadWrite", "offline_access"}, // You need to request offline_access in order to retrieve a refresh token
	}, bindPort, c.logger)
	if err != nil {
		return err
	}

	c.oAuthHandler = oAuthListener
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

		// this only checks the expiry field, which is the expiration time of the access token which was granted
		// even if the refresh token is still valid
		// TODO: unfortunately, without this part - the token will get assigned below and this triggers a panic
		// TODO: in the oauth2 package. I'm not aware of the culprit yet.
		now := time.Now()
		if now.After(expiry) {
			c.logger.Infof("saved credentials expired, we need to reauthenticate..")
			c.authenticated = false
			err := c.storage.RemoveCalendarAuth(c.calendarID)
			if err != nil {
				return fmt.Errorf("failed to remove authentication for calendar %s: %w", c.calendarID, err)
			}
			return nil
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

	client := c.oAuthConfig.Client(ctx, c.oAuthToken)

	resp, err := client.Get(baseUrl + "/me/calendars/" + c.calendarID)
	if err != nil {
		if strings.Contains(err.Error(), "token_expired") {
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
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code %d", resp.StatusCode)
	}
	c.outlookClient = &OutlookClient{Client: client, CalendarID: c.calendarID}
	return nil
}

func (c *CalendarAPI) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	events, err := c.outlookClient.ListEvents(ctx, start, end)
	if err != nil {
		return nil, err
	}

	log.WithFields(log.Fields{
		"adapter": c.Name(),
	}).Printf("loaded %d events between %s and %s.", len(events), start.Format(time.RFC1123), end.Format(time.RFC1123))

	return events, nil
}

func (c *CalendarAPI) CreateEvent(ctx context.Context, e models.Event) error {
	err := c.outlookClient.CreateEvent(ctx, e)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"adapter": c.Name(),
	}).Printf("Event %s at %s created", e.ShortTitle(), e.StartTime.String())

	return nil
}

func (c *CalendarAPI) UpdateEvent(ctx context.Context, e models.Event) error {
	err := c.outlookClient.UpdateEvent(ctx, e)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"adapter": c.Name(),
	}).Printf("Event %s at %s updated", e.ShortTitle(), e.StartTime.String())

	return nil
}

func (c *CalendarAPI) DeleteEvent(ctx context.Context, e models.Event) error {
	err := c.outlookClient.DeleteEvent(ctx, e)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"adapter": c.Name(),
	}).Printf("Event %s at %s deleted", e.ShortTitle(), e.StartTime.String())

	return nil
}

func (c *CalendarAPI) GetSourceID() string {
	return c.outlookClient.GetSourceID()
}

func (c *CalendarAPI) SetLogger(logger *log.Entry) {
	c.logger = logger
}

func (c *CalendarAPI) Name() string {
	return "Outlook"
}
