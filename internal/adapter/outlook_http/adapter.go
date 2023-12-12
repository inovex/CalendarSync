package outlook_http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/pkg/browser"
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
	}, bindPort)
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
		log.Debugf("stored access token expires on: %s", storedAuth.OAuth2.Expiry)
		expiry, err := time.Parse(time.RFC3339, storedAuth.OAuth2.Expiry)
		if err != nil {
			return err
		}

		now := time.Now()
		// If the access token is expired
		if now.After(expiry) {

			// trying to get a new access and refresh token using the refresh token
			c.logger.Info("stored access token expired, lets try to get a new one..")
			newAuthData, err := getNewRefreshToken(storedAuth.OAuth2.RefreshToken, credentials.Client.Id, endpoint.TokenURL)
			if err != nil {
				c.logger.Info("couldn't get a new access and refresh token, maybe the refresh token is also expired..")
				c.authenticated = false
				err = c.storage.RemoveCalendarAuth(c.calendarID)
				if err != nil {
					return fmt.Errorf("failed to remove authentication for calendar %s: %w", c.calendarID, err)
				}
				return nil
			}

			// we got a new access token, yay! 🎉
			newAuthExpiry, err := time.Parse(time.RFC3339, newAuthData.Expiry)
			if err != nil {
				return err
			}

			c.oAuthToken = &oauth2.Token{
				AccessToken:  newAuthData.AccessToken,
				RefreshToken: newAuthData.RefreshToken,
				Expiry:       newAuthExpiry,
				TokenType:    newAuthData.TokenType,
			}
			c.authenticated = true

			c.logger.Debugf("Got a new access token, which expires on: %s", c.oAuthToken.Expiry.Format(time.RFC3339))
			c.logger.Info("Refreshed credentials")

			return nil
		}

		// if the access token isn't expired, we can use our stored Credentials
		c.oAuthToken = &oauth2.Token{
			AccessToken:  storedAuth.OAuth2.AccessToken,
			RefreshToken: storedAuth.OAuth2.RefreshToken,
			Expiry:       expiry,
			TokenType:    storedAuth.OAuth2.TokenType,
		}

		c.authenticated = true
		c.logger.Info("stored credentials are still valid, we will use those")
	}

	return nil
}

func (c *CalendarAPI) Initialize(ctx context.Context, config map[string]interface{}) error {
	if !c.authenticated {
		c.oAuthUrl = c.oAuthHandler.Configuration().AuthCodeURL("state", oauth2.AccessTypeOffline)
		c.logger.Infof("opening browser window for authentication of %s\n", c.Name())
		err := browser.OpenURL(c.oAuthUrl)
		if err != nil {
			c.logger.Infof("browser did not open, please authenticate adapter %s:\n\n %s\n\n\n", c.Name(), c.oAuthUrl)
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
		c.logger.Debug("adapter is already authenticated, loading access token")
	}

	client := c.oAuthConfig.Client(ctx, c.oAuthToken)

	resp, err := client.Get(baseUrl + "/me/calendars/" + c.calendarID)
	if err != nil {
		if strings.Contains(err.Error(), "token_expired") {
			c.logger.Info("the refresh token expired, initiating reauthentication...")
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

// getNewRefreshToken gets a new access and refresh token from microsoft in exchange for a still valid refresh token and the clientID
func getNewRefreshToken(currentRefreshToken string, clientID string, tokenUrl string) (a auth.OAuth2Object, e error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", currentRefreshToken)
	data.Set("client_id", clientID)

	req, err := http.NewRequest("POST", tokenUrl, ioutil.NopCloser(strings.NewReader(data.Encode())))
	if err != nil {
		return a, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return a, err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return a, err
	}

	var tokenResponse TokenResponse

	err = json.Unmarshal(respBody, &tokenResponse)
	if err != nil {
		return a, err
	}

	a.AccessToken = tokenResponse.AccessToken
	a.RefreshToken = tokenResponse.RefreshToken
	a.TokenType = tokenResponse.TokenType
	a.Expiry = time.Now().Add(time.Duration(time.Second * time.Duration(tokenResponse.ExpiresIn))).Format(time.RFC3339)

	return a, nil
}
