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
)

// stubTransport returns a fixed status code and body for every request.
type stubTransport struct {
	status int
	body   string
}

func (s stubTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: s.status,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

// An Outlook event's city/state/postcode live only in the structured
// location.address object, while displayName is often just the street or a
// venue name. Mapping displayName alone dropped the address, so the synced
// copy carried only the street and navigation resolved it to a same-named
// street in the wrong city.
//
// Fixture is a real Graph payload: displayName is the bare street and
// Medina/Minnesota exist only inside address.
func TestListEvents_StructuredAddress_SurvivesMapping(t *testing.T) {
	graphResponse := `{"value":[{
		"id":"evt-arrowhead",
		"iCalUId":"evt-arrowhead-uid",
		"subject":"Meeting to discuss CRM",
		"start":{"dateTime":"2026-05-20T15:00:00.0000000","timeZone":"UTC"},
		"end":{"dateTime":"2026-05-20T16:00:00.0000000","timeZone":"UTC"},
		"location":{
			"displayName":"3640 Arrowhead Dr",
			"address":{
				"street":"3640 Arrowhead Dr",
				"city":"Medina",
				"state":"Minnesota",
				"postalCode":"55340",
				"countryOrRegion":"United States"
			}
		},
		"extensions":[]
	}]}`

	client := &OutlookClient{
		Client:     &http.Client{Transport: stubTransport{status: http.StatusOK, body: graphResponse}},
		CalendarID: "test-calendar",
	}

	events, err := client.ListEvents(context.Background(), time.Now(), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, events, 1)

	loc := events[0].Location
	assert.Contains(t, loc, "Medina", "city lost — navigation routes to the wrong city (got %q)", loc)
	assert.Contains(t, loc, "Minnesota", "state lost (got %q)", loc)
	// displayName ("3640 Arrowhead Dr") equals address.street here, so the
	// flatten must not repeat it.
	assert.Equal(t, 1, strings.Count(loc, "3640 Arrowhead Dr"), "street double-printed (got %q)", loc)
}
