package outlook_http

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/inovex/CalendarSync/internal/models"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestOutlookClientCalendarPath(t *testing.T) {
	tests := []struct {
		name         string
		user         string
		expectedPath string
	}{
		{
			name:         "current user",
			expectedPath: "/me/calendars/calendar-id",
		},
		{
			name:         "shared mailbox",
			user:         "shared-mailbox@example.com",
			expectedPath: "/users/shared-mailbox@example.com/calendars/calendar-id",
		},
		{
			name:         "escaped user path segment",
			user:         "shared/mailbox@example.com",
			expectedPath: "/users/shared%2Fmailbox@example.com/calendars/calendar-id",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := OutlookClient{CalendarID: "calendar-id", User: test.user}
			request, err := http.NewRequest(http.MethodGet, "https://example.com"+client.calendarPath(), nil)
			require.NoError(t, err)

			assert.Equal(t, test.expectedPath, request.URL.EscapedPath())
		})
	}
}

func TestOutlookClientRequestsUseCalendarPath(t *testing.T) {
	start := time.Date(2026, time.July, 20, 8, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	event := models.Event{
		ID:        "event-id",
		StartTime: start,
		EndTime:   end,
		Metadata:  models.NewEventMetadata("sync-id", "event-uri", "source-id"),
	}

	users := []struct {
		name     string
		user     string
		basePath string
	}{
		{
			name:     "current user",
			basePath: "/v1.0/me/calendars/calendar-id",
		},
		{
			name:     "shared mailbox",
			user:     "shared-mailbox@example.com",
			basePath: "/v1.0/users/shared-mailbox@example.com/calendars/calendar-id",
		},
	}
	operations := []struct {
		name           string
		method         string
		suffix         string
		responseStatus int
		responseBody   string
		invoke         func(*OutlookClient) error
	}{
		{
			name:           "list",
			method:         http.MethodGet,
			suffix:         "/CalendarView",
			responseStatus: http.StatusOK,
			responseBody:   `{"value":[]}`,
			invoke: func(client *OutlookClient) error {
				_, err := client.ListEvents(context.Background(), start, end)
				return err
			},
		},
		{
			name:           "create",
			method:         http.MethodPost,
			suffix:         "/events",
			responseStatus: http.StatusCreated,
			invoke: func(client *OutlookClient) error {
				return client.CreateEvent(context.Background(), event)
			},
		},
		{
			name:           "update",
			method:         http.MethodPatch,
			suffix:         "/events/event-id",
			responseStatus: http.StatusOK,
			responseBody:   `{}`,
			invoke: func(client *OutlookClient) error {
				return client.UpdateEvent(context.Background(), event)
			},
		},
		{
			name:           "delete",
			method:         http.MethodDelete,
			suffix:         "/events/event-id",
			responseStatus: http.StatusNoContent,
			invoke: func(client *OutlookClient) error {
				return client.DeleteEvent(context.Background(), event)
			},
		},
	}

	for _, user := range users {
		for _, operation := range operations {
			t.Run(user.name+" "+operation.name, func(t *testing.T) {
				requestSeen := false
				httpClient := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					requestSeen = true
					assert.Equal(t, operation.method, request.Method)
					assert.Equal(t, user.basePath+operation.suffix, request.URL.EscapedPath())

					if operation.name == "list" {
						assert.Equal(t, start.Format(timeFormat), request.URL.Query().Get("startDateTime"))
						assert.Equal(t, end.Format(timeFormat), request.URL.Query().Get("endDateTime"))
						assert.Equal(t, "extensions($filter=Id eq 'inovex.calendarsync.meta')", request.URL.Query().Get("$expand"))
						assert.Equal(t, `outlook.timezone="UTC"`, request.Header.Get("Prefer"))
					}
					if operation.name == "create" || operation.name == "update" {
						assert.Equal(t, "application/json", request.Header.Get("Content-Type"))
					}

					return &http.Response{
						StatusCode: operation.responseStatus,
						Body:       io.NopCloser(strings.NewReader(operation.responseBody)),
						Header:     make(http.Header),
					}, nil
				})}
				client := &OutlookClient{Client: httpClient, CalendarID: "calendar-id", User: user.user}

				require.NoError(t, operation.invoke(client))
				assert.True(t, requestSeen)
			})
		}
	}
}

func TestOutlookClientCalendarHash(t *testing.T) {
	currentUser := OutlookClient{CalendarID: "calendar-id"}
	sharedMailbox := OutlookClient{CalendarID: "calendar-id", User: "shared-mailbox@example.com"}
	otherMailbox := OutlookClient{CalendarID: "calendar-id", User: "other-mailbox@example.com"}

	assert.Equal(t, "lL43FOf3Lx8yhz-dV1N6PM3pRSc=", currentUser.GetCalendarHash())
	assert.NotEqual(t, currentUser.GetCalendarHash(), sharedMailbox.GetCalendarHash())
	assert.NotEqual(t, sharedMailbox.GetCalendarHash(), otherMailbox.GetCalendarHash())
	assert.Equal(t, sharedMailbox.GetCalendarHash(), sharedMailbox.GetCalendarHash())
}
