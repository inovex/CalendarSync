package outlook_http

import (
	"context"
	"encoding/json"
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
		Metadata: &models.Metadata{
			SyncID:           "sync-id",
			OriginalEventUri: "event-uri",
			SourceID:         "source-id",
		},
	}

	users := []struct {
		name           string
		user           string
		basePath       string
		expectedExpand string
	}{
		{
			name:           "current user",
			basePath:       "/v1.0/me/calendars/calendar-id",
			expectedExpand: "extensions($filter=Id eq 'inovex.calendarsync.meta')",
		},
		{
			name:           "shared mailbox",
			user:           "shared-mailbox@example.com",
			basePath:       "/v1.0/users/shared-mailbox@example.com/calendars/calendar-id",
			expectedExpand: "singleValueExtendedProperties($filter=id eq 'String {23f8dbef-16e9-5e2c-8cc7-e7f020136a50} Name inovex.calendarsync.meta')",
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
						assert.Equal(t, user.expectedExpand, request.URL.Query().Get("$expand"))
						assert.Equal(t, `outlook.timezone="UTC"`, request.Header.Get("Prefer"))
					}
					if operation.name == "create" || operation.name == "update" {
						assert.Equal(t, "application/json", request.Header.Get("Content-Type"))

						body, err := io.ReadAll(request.Body)
						require.NoError(t, err)
						var payload map[string]interface{}
						require.NoError(t, json.Unmarshal(body, &payload))

						if user.user == "" {
							require.NotContains(t, payload, "singleValueExtendedProperties")
							require.Contains(t, payload, "extensions")
							extensions := payload["extensions"].([]interface{})
							require.Len(t, extensions, 1)
							extension := extensions[0].(map[string]interface{})
							assert.Equal(t, "inovex.calendarsync.meta", extension["extensionName"])
							assert.Equal(t, "sync-id", extension["SyncID"])
							assert.Equal(t, "event-uri", extension["OriginalEventUri"])
							assert.Equal(t, "source-id", extension["SourceID"])
						} else {
							require.NotContains(t, payload, "extensions")
							require.Contains(t, payload, "singleValueExtendedProperties")
							properties := payload["singleValueExtendedProperties"].([]interface{})
							require.Len(t, properties, 1)
							property := properties[0].(map[string]interface{})
							assert.Equal(t, "String {23f8dbef-16e9-5e2c-8cc7-e7f020136a50} Name inovex.calendarsync.meta", property["id"])

							var metadata map[string]string
							require.NoError(t, json.Unmarshal([]byte(property["value"].(string)), &metadata))
							assert.Equal(t, map[string]string{
								"SyncID":           "sync-id",
								"OriginalEventUri": "event-uri",
								"SourceID":         "source-id",
							}, metadata)
						}
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

func TestOutlookClientReadsSingleValueMetadata(t *testing.T) {
	outlookEvent := Event{
		ID:       "event-id",
		HtmlLink: "https://example.com/event",
		Start:    Time{DateTime: "2026-07-20T08:00:00.0000000", TimeZone: "UTC"},
		End:      Time{DateTime: "2026-07-20T09:00:00.0000000", TimeZone: "UTC"},
		SingleValueExtendedProperties: []SingleValueExtendedProperty{
			{
				ID:    "String {23f8dbef-16e9-5e2c-8cc7-e7f020136a50} Name inovex.calendarsync.meta",
				Value: `{"SyncID":"sync-id","OriginalEventUri":"event-uri","SourceID":"source-id"}`,
			},
		},
	}

	event, err := (OutlookClient{}).outlookEventToEvent(outlookEvent, "adapter-source-id")

	require.NoError(t, err)
	assert.Equal(t, &models.Metadata{
		SyncID:           "sync-id",
		OriginalEventUri: "event-uri",
		SourceID:         "source-id",
	}, event.Metadata)
}

func TestOutlookClientRejectsInvalidSingleValueMetadata(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "malformed JSON", value: `{not-json}`},
		{name: "missing source identifier", value: `{"SyncID":"sync-id"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			outlookEvent := Event{
				ID:    "event-id",
				Start: Time{DateTime: "2026-07-20T08:00:00.0000000", TimeZone: "UTC"},
				End:   Time{DateTime: "2026-07-20T09:00:00.0000000", TimeZone: "UTC"},
				SingleValueExtendedProperties: []SingleValueExtendedProperty{
					{
						ID:    "String {23f8dbef-16e9-5e2c-8cc7-e7f020136a50} Name inovex.calendarsync.meta",
						Value: test.value,
					},
				},
			}

			_, err := (OutlookClient{}).outlookEventToEvent(outlookEvent, "adapter-source-id")

			require.ErrorContains(t, err, "cannot decode Outlook event metadata")
		})
	}
}

func TestOutlookClientReturnsGraphErrors(t *testing.T) {
	start := time.Date(2026, time.July, 20, 8, 0, 0, 0, time.UTC)
	event := models.Event{ID: "event-id"}
	tests := []struct {
		name   string
		invoke func(*OutlookClient) error
	}{
		{
			name: "list",
			invoke: func(client *OutlookClient) error {
				_, err := client.ListEvents(context.Background(), start, start.Add(time.Hour))
				return err
			},
		},
		{
			name: "delete",
			invoke: func(client *OutlookClient) error {
				return client.DeleteEvent(context.Background(), event)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &OutlookClient{
				CalendarID: "calendar-id",
				Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusForbidden,
						Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"ErrorAccessDenied","message":"Access is denied."}}`)),
						Header:     make(http.Header),
					}, nil
				})},
			}

			err := test.invoke(client)

			require.ErrorContains(t, err, "status code 403")
			require.ErrorContains(t, err, "ErrorAccessDenied")
		})
	}
}

func TestOutlookClientReturnsGraphErrorFromNextPage(t *testing.T) {
	requestCount := 0
	client := &OutlookClient{
		CalendarID: "calendar-id",
		Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			requestCount++
			if requestCount == 1 {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"@odata.nextLink":"https://graph.microsoft.com/next","value":[]}`)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusForbidden,
				Body:       io.NopCloser(strings.NewReader(`{"error":{"code":"ErrorAccessDenied","message":"Access is denied."}}`)),
				Header:     make(http.Header),
			}, nil
		})},
	}

	_, err := client.ListEvents(
		context.Background(),
		time.Date(2026, time.July, 20, 8, 0, 0, 0, time.UTC),
		time.Date(2026, time.July, 20, 9, 0, 0, 0, time.UTC),
	)

	require.ErrorContains(t, err, "status code 403")
	require.ErrorContains(t, err, "ErrorAccessDenied")
}
