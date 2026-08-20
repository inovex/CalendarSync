package outlook_http

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/inovex/CalendarSync/internal/auth"
	"github.com/inovex/CalendarSync/internal/config"
)

type memoryStorage struct{}

func (memoryStorage) WriteCalendarAuth(auth.CalendarAuth) (bool, error) {
	return true, nil
}

func (memoryStorage) ReadCalendarAuth(string) (*auth.CalendarAuth, error) {
	return nil, nil
}

func (memoryStorage) RemoveCalendarAuth(string) error {
	return nil
}

func (memoryStorage) Setup(config.AuthStorage, string) error {
	return nil
}

func TestCalendarAPISetConfig(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectedUser  string
		expectedError string
	}{
		{
			name: "absent user",
		},
		{
			name:   "empty user",
			config: map[string]interface{}{"user": ""},
		},
		{
			name:         "shared mailbox",
			config:       map[string]interface{}{"user": "shared-mailbox@example.com"},
			expectedUser: "shared-mailbox@example.com",
		},
		{
			name:          "non-string user",
			config:        map[string]interface{}{"user": 42},
			expectedError: "Outlook adapter config 'user' must be a string",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calendarAPI := &CalendarAPI{}

			err := calendarAPI.SetConfig(test.config)

			if test.expectedError != "" {
				require.EqualError(t, err, test.expectedError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.expectedUser, calendarAPI.user)
		})
	}
}

func TestCalendarAPIAuthorizationURLScopes(t *testing.T) {
	tests := []struct {
		name           string
		config         map[string]interface{}
		expectedScopes []string
	}{
		{
			name:           "current user",
			expectedScopes: []string{"Calendars.ReadWrite", "offline_access"},
		},
		{
			name:           "shared mailbox",
			config:         map[string]interface{}{"user": "shared-mailbox@example.com"},
			expectedScopes: []string{"Calendars.ReadWrite", "Calendars.ReadWrite.Shared", "offline_access"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calendarAPI := &CalendarAPI{calendarID: "calendar-id"}
			require.NoError(t, calendarAPI.SetConfig(test.config))

			err := calendarAPI.SetupOauth2(
				context.Background(),
				auth.Credentials{
					Client: auth.Client{Id: "client-id"},
					Tenant: auth.Tenant{Id: "tenant-id"},
				},
				memoryStorage{},
				0,
			)
			require.NoError(t, err)

			authorizationURL, err := url.Parse(calendarAPI.oAuthUrl)
			require.NoError(t, err)
			urlScopes := strings.Fields(authorizationURL.Query().Get("scope"))

			assert.ElementsMatch(t, test.expectedScopes, urlScopes)
			assert.ElementsMatch(t, test.expectedScopes, calendarAPI.oAuthConfig.Scopes)
		})
	}
}
