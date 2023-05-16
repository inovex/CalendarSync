package zep

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/inovex/CalendarSync/internal/models"

	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
	log "github.com/sirupsen/logrus"
)

const (
	endpointKey = "endpoint"
	usernameKey = "username"
	passwordKey = "password"
)

type CalendarAPI struct {
	username string
	password string
	endpoint string

	client *caldav.Client
	logger *log.Entry

	calendarID string

	principal string
	homeSet   string
}

func (zep *CalendarAPI) GetSourceID() string {
	return zep.generateSourceID()
}

func (zep *CalendarAPI) generateSourceID() string {
	var id []byte
	components := []string{zep.username, zep.homeSet, zep.calendarID}

	sum := sha1.Sum([]byte(strings.Join(components, "")))
	id = append(id, sum[:]...)
	return base64.URLEncoding.EncodeToString(id)

}
func (zep *CalendarAPI) SetLogger(logger *log.Entry) {
	zep.logger = logger
}

func (zep *CalendarAPI) Name() string {
	return "ZEP CalDav API"
}

func (zep *CalendarAPI) Initialize(ctx context.Context, calendarID string, config map[string]interface{}) error {
	if _, ok := config[usernameKey]; !ok {
		return fmt.Errorf("missing config key: %s", usernameKey)
	}
	if _, ok := config[passwordKey]; !ok {
		return fmt.Errorf("missing config key: %s", passwordKey)
	}
	if _, ok := config[endpointKey]; !ok {
		return fmt.Errorf("missing config key: %s", endpointKey)
	}
	zep.username = config[usernameKey].(string)
	zep.password = config[passwordKey].(string)
	zep.endpoint = config[endpointKey].(string)

	var err error
	httpClient := webdav.HTTPClientWithBasicAuth(http.DefaultClient, zep.username, zep.password)
	zep.client, err = caldav.NewClient(httpClient, zep.endpoint)
	if err != nil {
		return fmt.Errorf("unable to create caldav client: %w", err)
	}

	// finds the principal path of the user
	zep.principal, err = zep.client.FindCurrentUserPrincipal()
	if err != nil {
		return fmt.Errorf("unable to find user principal: %w", err)
	}

	// finds the base path to the users calendars
	zep.homeSet, err = zep.client.FindCalendarHomeSet(zep.principal)
	if err != nil {
		return fmt.Errorf("unable to find calendar homeSet: %w", err)
	}
	zep.logger = log.New().WithField("client", "zep")
	zep.calendarID = calendarID
	return nil
}

func (zep *CalendarAPI) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	absences, err := zep.ListEvents(start, end)
	if err != nil {
		return nil, fmt.Errorf("could not get absences %w", err)
	}

	var syncEvents []models.Event
	for _, v := range absences {
		syncEvents = append(syncEvents,
			models.Event{
				ICalUID:     v.ID,
				Title:       v.Summary,
				Description: v.Description,
				StartTime:   v.Start,
				EndTime:     v.End,
				Metadata:    models.NewEventMetadata(v.ID, "", zep.GetSourceID()),
			})
	}

	return syncEvents, nil
}

// ListEvents returns all events of the given calendar of a user (if it exists).
func (zep *CalendarAPI) ListEvents(from, to time.Time) ([]Event, error) {
	calendars, err := zep.client.FindCalendars(zep.homeSet)
	if err != nil {
		return nil, fmt.Errorf("cannot find calendars: %w", err)
	}

	var eventCalendar caldav.Calendar
	for _, calendar := range calendars {
		if strings.Contains(calendar.Path, zep.calendarID) {
			eventCalendar = calendar
		}
	}
	if eventCalendar == (caldav.Calendar{}) {
		return nil, fmt.Errorf("no %s calendar found in %s", zep.calendarID, zep.homeSet)
	}

	// query all events inside the given time range (inclusive)
	ret, err := zep.client.QueryCalendar(eventCalendar.Path, &caldav.CalendarQuery{
		CompRequest: caldav.CalendarCompRequest{
			Name:     "VCALENDAR",
			AllProps: true,
			Props:    nil,
			AllComps: false,
			Comps:    []caldav.CalendarCompRequest{},
		},
		CompFilter: caldav.CompFilter{
			Name:  "VCALENDAR",
			Start: time.Time{},
			End:   time.Time{},
			Props: nil,
			Comps: []caldav.CompFilter{
				{
					Name:  "VEVENT",
					Start: from,
					End:   to,
				},
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("unable to query calendar %s: %w", eventCalendar.Path, err)
	}

	var events []Event

	// start deconstructing the events if there are any
	for _, object := range ret {
		if len(object.Data.Events()) > 0 {

			for _, calDavEvent := range object.Data.Events() {
				event, err := eventFromCalDavEvent(calDavEvent, object.ETag)
				if err != nil {
					// todo: handle properly
					log.Errorln(err)
					continue
				}
				events = append(events, event)
			}
		}
	}

	return events, nil
}

// todo: read timezone from event, not just assume time.Local
func eventFromCalDavEvent(event ical.Event, etag string) (Event, error) {
	start, err := event.Component.Props.Get("dtstart").DateTime(time.Local)
	if err != nil {
		return Event{}, fmt.Errorf("unable to decode dtstart: %w", err)
	}

	end, err := event.Component.Props.Get("dtend").DateTime(time.Local)
	if err != nil {
		return Event{}, fmt.Errorf("unable to decode dtend: %w", err)
	}

	return Event{
		ID:          event.Component.Props.Get("uid").Value,
		Start:       start,
		End:         end,
		Summary:     safeGetComponentPropValueString(event, "summary"),
		Category:    safeGetComponentPropValueString(event, "categories"),
		Description: safeGetComponentPropValueString(event, "description"),
		Etag:        etag,
	}, nil
}

func safeGetComponentPropValueString(event ical.Event, key string) string {
	prop := event.Component.Props.Get(key)
	if prop == nil {
		return ""
	}
	return prop.Value
}
