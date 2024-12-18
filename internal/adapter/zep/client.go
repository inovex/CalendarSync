package zep

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/inovex/CalendarSync/internal/adapter/port"
	"github.com/inovex/CalendarSync/internal/models"

	"github.com/charmbracelet/log"
	"github.com/emersion/go-ical"
	"github.com/emersion/go-webdav"
	"github.com/emersion/go-webdav/caldav"
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

	calendarID string
	logger     *log.Logger

	principal string
	homeSet   string
}

// Assert that the expected interfaces are implemented
var _ port.Configurable = &CalendarAPI{}
var _ port.LogSetter = &CalendarAPI{}
var _ port.CalendarIDSetter = &CalendarAPI{}

func (zep *CalendarAPI) SetCalendarID(calendarID string) error {
	if calendarID == "" {
		return fmt.Errorf("%s adapter 'calendar' cannot be empty", zep.Name())
	}
	zep.calendarID = calendarID
	return nil
}

func (zep *CalendarAPI) GetCalendarHash() string {
	var id []byte
	components := []string{zep.username, zep.homeSet, zep.calendarID}

	sum := sha1.Sum([]byte(strings.Join(components, "")))
	id = append(id, sum[:]...)
	return base64.URLEncoding.EncodeToString(id)
}

func (zep *CalendarAPI) Name() string {
	return "ZEP CalDav API"
}

func (zep *CalendarAPI) Initialize(ctx context.Context, openBrowser bool, config map[string]interface{}) error {
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
	zep.principal, err = zep.client.FindCurrentUserPrincipal(context.Background())
	if err != nil {
		return fmt.Errorf("unable to find user principal: %w", err)
	}

	// finds the base path to the users calendars
	zep.homeSet, err = zep.client.FindCalendarHomeSet(context.Background(), zep.principal)
	if err != nil {
		return fmt.Errorf("unable to find calendar homeSet: %w", err)
	}
	return nil
}

func (zep *CalendarAPI) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	events, err := zep.ListEvents(start, end)
	if err != nil {
		return nil, fmt.Errorf("could not get zep events %w", err)
	}

	var syncEvents []models.Event

	zep.logger.Infof("loaded %d events between %s and %s.", len(events), start.Format(time.DateOnly), end.Format(time.DateOnly))

	for _, v := range events {
		syncEvents = append(syncEvents,
			models.Event{
				ICalUID:     v.ID,
				Title:       v.Summary,
				Description: v.Description,
				StartTime:   v.Start,
				EndTime:     v.End,
				AllDay:      v.AllDay,
				Accepted:    true,
				Metadata:    models.NewEventMetadata(v.ID, "", zep.GetCalendarHash()),
			})
	}

	return syncEvents, nil
}

// ListEvents returns all events of the given calendar of a user (if it exists).
func (zep *CalendarAPI) ListEvents(from, to time.Time) ([]Event, error) {
	calendars, err := zep.client.FindCalendars(context.Background(), zep.homeSet)
	if err != nil {
		return nil, fmt.Errorf("cannot find calendars: %w", err)
	}

	var eventCalendar caldav.Calendar
	for _, calendar := range calendars {
		if strings.Contains(calendar.Path, zep.calendarID) {
			eventCalendar = calendar
		}
	}

	// query all events inside the given time range (inclusive)
	ret, err := zep.client.QueryCalendar(context.Background(), eventCalendar.Path, &caldav.CalendarQuery{
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
					log.Error(err)
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
	start, err := event.Props.Get("dtstart").DateTime(time.Local)
	if err != nil {
		return Event{}, fmt.Errorf("unable to decode dtstart: %w", err)
	}

	end, err := event.Props.Get("dtend").DateTime(time.Local)
	if err != nil {
		return Event{}, fmt.Errorf("unable to decode dtend: %w", err)
	}

	// if it is an all-day event, all the times are zero
	allDay := (start.Hour() == 0 && start.Minute() == 0 && start.Second() == 0 &&
		end.Hour() == 0 && end.Minute() == 0 && end.Second() == 0)

	return Event{
		ID:          event.Props.Get("uid").Value,
		Start:       start,
		End:         end,
		AllDay:      allDay,
		Summary:     safeGetComponentPropValueString(event, "summary"),
		Category:    safeGetComponentPropValueString(event, "categories"),
		Description: safeGetComponentPropValueString(event, "description"),
		Etag:        etag,
	}, nil
}

func safeGetComponentPropValueString(event ical.Event, key string) string {
	prop := event.Props.Get(key)
	if prop == nil {
		return ""
	}
	return prop.Value
}

func (zep *CalendarAPI) SetLogger(logger *log.Logger) {
	zep.logger = logger
}
