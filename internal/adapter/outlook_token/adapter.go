package outlook_token

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/browser"

	"github.com/inovex/CalendarSync/internal/adapter/port"
	"github.com/inovex/CalendarSync/internal/auth"
	"github.com/inovex/CalendarSync/internal/models"
)

const (
	graphUrl   = "https://developer.microsoft.com/en-us/graph/graph-explorer"
	baseUrl    = "https://graph.microsoft.com/v1.0"
	timeFormat = "2006-01-02T15:04:05.0000000"
)

type ROOutlookCalendarClient interface {
	ListEvents(ctx context.Context, starttime time.Time, endtime time.Time) ([]models.Event, error)
	GetCalendarHash() string
}

type ROCalendarAPI struct {
	outlookClient ROOutlookCalendarClient
	calendarID    string

	logger *log.Logger

	storage auth.Storage
}

// Assert that the expected interfaces are implemented
var _ port.Configurable = &ROCalendarAPI{}
var _ port.LogSetter = &ROCalendarAPI{}
var _ port.CalendarIDSetter = &ROCalendarAPI{}
var _ port.StorageSetter = &ROCalendarAPI{}

func (c *ROCalendarAPI) SetCalendarID(calendarID string) error {
	if calendarID == "" {
		return fmt.Errorf("%s adapter 'calendar' cannot be empty", c.Name())
	}
	c.calendarID = calendarID
	return nil
}

func (c *ROCalendarAPI) Initialize(ctx context.Context, openBrowser bool, config map[string]interface{}) error {
	storedAuth, err := c.storage.ReadCalendarAuth(c.calendarID)
	if err != nil {
		return err
	}
	accessToken := ""
	if storedAuth != nil && storedAuth.AccessToken.Expiry.After(time.Now()) {
		c.logger.Debug("adapter is already authenticated, loading access token")
		accessToken = storedAuth.AccessToken.AccessToken
	} else {
		if openBrowser {
			c.logger.Infof("opening browser window for authentication of %s\n", c.Name())
			err := browser.OpenURL(graphUrl)
			if err != nil {
				c.logger.Infof("browser did not open, please authenticate adapter %s:\n\n %s\n\n\n", c.Name(), graphUrl)
			}
		} else {
			c.logger.Infof("Please authenticate adapter %s:\n\n %s\n\n\n", c.Name(), graphUrl)
		}
		fmt.Print("Copy access token from \"Access token\" tab: ")
		tokenString := ""
		fmt.Scanf("%s", &tokenString)
		jwtParser := jwt.Parser{}
		token, _, err := jwtParser.ParseUnverified(tokenString, &jwt.RegisteredClaims{})

		if err != nil {
			return err
		}

		expirationTime, err := token.Claims.GetExpirationTime()
		if err != nil {
			return err
		}

		if expirationTime.Time.Before(time.Now()) {
			return errors.New("Access token expired")
		}

		c.logger.Infof("access token valid until %v", expirationTime.Time.Format(time.RFC1123))

		accessToken = tokenString
		_, err = c.storage.WriteCalendarAuth(auth.CalendarAuth{
			CalendarID: c.calendarID,
			AccessToken: auth.AccessTokenObject{
				AccessToken: accessToken,
				Expiry:      expirationTime.Time,
			},
		})
		if err != nil {
			return err
		}
		c.logger.Debugf("access token stored successfully")
	}

	client := &http.Client{}
	c.outlookClient = &ROOutlookClient{Client: client, AccessToken: accessToken, CalendarID: c.calendarID}
	return nil
}

func (c *ROCalendarAPI) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	events, err := c.outlookClient.ListEvents(ctx, start, end)
	if err != nil {
		return nil, err
	}

	c.logger.Infof("loaded %d events between %s and %s.", len(events), start.Format(time.RFC1123), end.Format(time.RFC1123))

	return events, nil
}

func (c *ROCalendarAPI) GetCalendarHash() string {
	return c.outlookClient.GetCalendarHash()
}

func (c *ROCalendarAPI) Name() string {
	return "Outlook"
}

func (c *ROCalendarAPI) SetLogger(logger *log.Logger) {
	c.logger = logger
}

func (c *ROCalendarAPI) SetStorage(storage auth.Storage) {
	c.storage = storage
}
