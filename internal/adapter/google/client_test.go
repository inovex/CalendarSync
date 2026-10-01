package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// newPagedClient returns a GCalClient talking to a fake Calendar API which serves the given pages in order.
func newPagedClient(t *testing.T, pages [][]string) *GCalClient {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := 0
		if token := r.URL.Query().Get("pageToken"); token != "" {
			if !assert.NoError(t, json.Unmarshal([]byte(token), &page)) {
				http.Error(w, "invalid page token", http.StatusBadRequest)
				return
			}
		}

		if page < 0 || page >= len(pages) {
			t.Errorf("unexpected page index: %d", page)
			http.Error(w, "unexpected page", http.StatusBadRequest)
			return
		}

		var items []map[string]any
		for _, id := range pages[page] {
			items = append(items, map[string]any{
				"id":        id,
				"summary":   "event " + id,
				"start":     map[string]string{"dateTime": "2026-10-01T10:00:00Z"},
				"end":       map[string]string{"dateTime": "2026-10-01T11:00:00Z"},
				"reminders": map[string]bool{"useDefault": true},
			})
		}
		response := map[string]any{"kind": "calendar#events", "items": items}
		if page+1 < len(pages) {
			next, _ := json.Marshal(page + 1)
			response["nextPageToken"] = string(next)
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(response))
	}))
	t.Cleanup(srv.Close)

	service, err := calendar.NewService(context.Background(), option.WithHTTPClient(srv.Client()), option.WithEndpoint(srv.URL+"/"))
	require.NoError(t, err)

	client := &GCalClient{Client: service, CalendarId: "test@example.com"}
	client.InitRateLimiter()
	return client
}

func TestListEvents_returnsEveryEventExactlyOnce(t *testing.T) {
	tt := []struct {
		name  string
		pages [][]string
		want  []string
	}{
		{
			name:  "empty calendar",
			pages: [][]string{{}},
		},
		{
			name:  "empty first page",
			pages: [][]string{{}, {"a"}},
			want:  []string{"a"},
		},
		{
			name:  "empty intermediate page",
			pages: [][]string{{"a"}, {}, {"b"}},
			want:  []string{"a", "b"},
		},
		{
			name:  "empty last page",
			pages: [][]string{{"a"}, {}},
			want:  []string{"a"},
		},
		{
			name:  "single page",
			pages: [][]string{{"a", "b"}},
			want:  []string{"a", "b"},
		},
		{
			name:  "two pages",
			pages: [][]string{{"a", "b"}, {"c"}},
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "three pages",
			pages: [][]string{{"a", "b"}, {"c", "d"}, {"e"}},
			want:  []string{"a", "b", "c", "d", "e"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			client := newPagedClient(t, tc.pages)

			start := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
			events, err := client.ListEvents(context.Background(), start, start.Add(24*time.Hour))
			require.NoError(t, err)

			var got []string
			for _, event := range events {
				got = append(got, event.ID)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
