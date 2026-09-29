package google

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/inovex/CalendarSync/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/ratelimit"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

func TestDeleteEvent(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		reason  string
		wantErr bool
	}{
		{name: "deleted", status: http.StatusNoContent},
		{name: "not found", status: http.StatusNotFound, reason: "notFound"},
		{name: "already deleted", status: http.StatusGone, reason: "deleted"},
		{name: "full sync required", status: http.StatusGone, reason: "fullSyncRequired", wantErr: true},
		{name: "old modification time", status: http.StatusGone, reason: "updatedMinTooLongAgo", wantErr: true},
		{name: "unknown gone response", status: http.StatusGone, wantErr: true},
		{name: "forbidden", status: http.StatusForbidden, reason: "forbidden", wantErr: true},
		{name: "server error", status: http.StatusInternalServerError, reason: "backendError", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/calendars/calendar/events/event", r.URL.Path)
				assert.Equal(t, "none", r.URL.Query().Get("sendUpdates"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				if tc.status != http.StatusNoContent {
					_, err := fmt.Fprintf(w, `{"error":{"code":%d,"message":"test response","errors":[{"reason":%q}]}}`, tc.status, tc.reason)
					assert.NoError(t, err)
				}
			}))
			t.Cleanup(server.Close)

			ctx := context.Background()
			service, err := calendar.NewService(ctx, option.WithHTTPClient(server.Client()), option.WithEndpoint(server.URL+"/"))
			require.NoError(t, err)
			logger := log.New(io.Discard)
			adapter := &CalendarAPI{
				gcalClient: &GCalClient{
					Client:      service,
					RateLimiter: ratelimit.NewUnlimited(),
					CalendarId:  "calendar",
					logger:      logger,
				},
				logger: logger,
			}

			err = adapter.DeleteEvent(ctx, models.Event{ID: "event", Title: "Meeting"})
			if tc.wantErr {
				var apiErr *googleapi.Error
				require.ErrorAs(t, err, &apiErr)
				assert.Equal(t, tc.status, apiErr.Code)
				assert.Equal(t, tc.reason, firstErrorReason(apiErr))
			} else {
				require.NoError(t, err)
			}
			assert.EqualValues(t, 1, requests.Load())
		})
	}
}
