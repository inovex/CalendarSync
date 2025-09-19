package outlook

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

type TokenCalendarAPI struct {
	outlookClient OutlookCalendarClient
	calendarID    string

	logger *log.Logger

	storage auth.Storage
}

// Assert that the expected interfaces are implemented
var _ port.Configurable = &TokenCalendarAPI{}
var _ port.LogSetter = &TokenCalendarAPI{}
var _ port.CalendarIDSetter = &TokenCalendarAPI{}
var _ port.StorageSetter = &TokenCalendarAPI{}

func (c *TokenCalendarAPI) SetCalendarID(calendarID string) error {
	if calendarID == "" {
		return fmt.Errorf("%s adapter 'calendar' cannot be empty", c.Name())
	}
	c.calendarID = calendarID
	return nil
}

type AddHeaderTransport struct {
	AccessToken string
	T           http.RoundTripper
}

func (adt *AddHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", "Bearer "+adt.AccessToken)
	return adt.T.RoundTrip(req)
}

func (c *TokenCalendarAPI) Initialize(ctx context.Context, openBrowser bool, config map[string]interface{}) error {
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

	client := &http.Client{Transport: &AddHeaderTransport{accessToken, http.DefaultTransport}}
	c.outlookClient = &OutlookClient{Client: client, CalendarID: c.calendarID}
	return nil
}

func (c *TokenCalendarAPI) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	events, err := c.outlookClient.ListEvents(ctx, start, end)
	if err != nil {
		return nil, err
	}

	c.logger.Infof("loaded %d events between %s and %s.", len(events), start.Format(time.RFC1123), end.Format(time.RFC1123))

	return events, nil
}

func (c *TokenCalendarAPI) CreateEvent(ctx context.Context, e models.Event) error {
	err := c.outlookClient.CreateEvent(ctx, e)
	if err != nil {
		return err
	}

	c.logger.Info("Event created", "title", e.ShortTitle(), "time", e.StartTime.String())

	return nil
}

func (c *TokenCalendarAPI) UpdateEvent(ctx context.Context, e models.Event) error {
	err := c.outlookClient.UpdateEvent(ctx, e)
	if err != nil {
		return err
	}

	c.logger.Info("Event updated", "title", e.ShortTitle(), "time", e.StartTime.String())

	return nil
}

func (c *TokenCalendarAPI) DeleteEvent(ctx context.Context, e models.Event) error {
	err := c.outlookClient.DeleteEvent(ctx, e)
	if err != nil {
		return err
	}

	c.logger.Info("Event deleted", "title", e.ShortTitle(), "time", e.StartTime.String())

	return nil
}

func (c *TokenCalendarAPI) GetCalendarHash() string {
	return c.outlookClient.GetCalendarHash()
}

func (c *TokenCalendarAPI) Name() string {
	return "Outlook"
}

func (c *TokenCalendarAPI) SetLogger(logger *log.Logger) {
	c.logger = logger
}

func (c *TokenCalendarAPI) SetStorage(storage auth.Storage) {
	c.storage = storage
}
