package google

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/inovex/CalendarSync/internal/models"

	"github.com/charmbracelet/log"
	"go.uber.org/ratelimit"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

const (
	defaultPageMaxResults = 250
	maxCallsPerSecond     = 10
)

// GCalClient implements the GoogleCalendarClient interface
type GCalClient struct {
	Client      *calendar.Service
	RateLimiter ratelimit.Limiter
	CalendarId  string
	oauthClient *http.Client
	logger      *log.Logger
}

func (g *GCalClient) InitGoogleCalendarClient(CalendarId string, logger *log.Logger) error {
	apiClient, err := calendar.NewService(context.Background(), option.WithHTTPClient(g.oauthClient))
	if err != nil {
		return fmt.Errorf("unable to retrieve Calendar client: %w", err)
	}
	g.Client = apiClient
	g.CalendarId = CalendarId
	g.logger = logger
	g.InitRateLimiter()
	return nil
}

func (g *GCalClient) InitRateLimiter() {
	g.RateLimiter = ratelimit.New(maxCallsPerSecond)
}

func (g *GCalClient) ListEvents(ctx context.Context, starttime time.Time, endtime time.Time) ([]models.Event, error) {
	g.RateLimiter.Take()
	listCall := g.Client.Events.List(g.CalendarId).
		ShowDeleted(false).
		SingleEvents(true).
		// see: https://developers.google.com/calendar/api/v3/reference/events/list
		EventTypes("default", "focusTime", "outOfOffice").
		TimeMin(starttime.Format(time.RFC3339)).
		TimeMax(endtime.Format(time.RFC3339)).
		MaxResults(defaultPageMaxResults).
		OrderBy("startTime").
		Context(ctx)

	// perform initial list call
	eventList, err := listCall.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list eventList from calendar %s: %w", g.CalendarId, err)
	}

	var loadedEvents []models.Event
	for _, event := range eventList.Items {
		loadedEvents = append(loadedEvents, calendarEventToEvent(event, g.GetCalendarID()))
	}

	// if the responses 'nextPageToken' is set, the result is paginated and more data to be loaded recursively
	if eventList.NextPageToken != "" {
		err := g.loadPages(listCall, &eventList.Items, eventList.NextPageToken)
		if err != nil {
			return nil, err
		}
		for _, pageEvent := range eventList.Items {
			loadedEvents = append(loadedEvents, calendarEventToEvent(pageEvent, g.GetCalendarID()))
		}
	}
	return loadedEvents, nil
}

func (g *GCalClient) CreateEvent(ctx context.Context, event models.Event) error {
	extProperties := &calendar.EventExtendedProperties{
		Private: eventMetadataToEventProperties(event.Metadata),
	}

	var calendarAttendees []*calendar.EventAttendee
	for _, attendee := range event.Attendees {
		calendarAttendees = append(calendarAttendees, &calendar.EventAttendee{
			Email:       attendee.Email,
			DisplayName: attendee.DisplayName,
		})
	}

	var calendarReminders calendar.EventReminders
	for _, reminder := range event.Reminders {
		if reminder.Actions == models.ReminderActionDisplay {
			calendarReminders.ForceSendFields = []string{"UseDefault"}
			calendarReminders.Overrides = append(calendarReminders.Overrides, &calendar.EventReminder{
				Method:  "popup",
				Minutes: int64(event.StartTime.Sub(reminder.Trigger.PointInTime).Minutes()),
			})
		}
	}

	call, err := retry(ctx, func() (*calendar.Event, error) {
		g.RateLimiter.Take()
		return g.Client.Events.Insert(g.CalendarId, &calendar.Event{
			Summary:            event.Title,
			Description:        event.Description,
			Location:           event.Location,
			Start:              timeToEventDateTime(event.AllDay, event.StartTime),
			End:                timeToEventDateTime(event.AllDay, event.EndTime),
			ExtendedProperties: extProperties,
			Attendees:          calendarAttendees,
			Reminders:          &calendarReminders,
		}).Context(ctx).SendUpdates("none").Do()
	})
	if err != nil {
		return err
	}
	by, err := call.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal insert call: %w", err)
	}
	g.logger.Debugf("Insert Call:\n%v", string(by))
	return nil
}

func isNotFound(err error) bool {
	var gerr *googleapi.Error
	return errors.As(err, &gerr) && gerr.Code == http.StatusNotFound
}

func (g *GCalClient) UpdateEvent(ctx context.Context, event models.Event) error {
	extProperties := &calendar.EventExtendedProperties{
		Private: eventMetadataToEventProperties(event.Metadata),
	}

	var calendarAttendees []*calendar.EventAttendee
	for _, attendee := range event.Attendees {
		calendarAttendees = append(calendarAttendees, &calendar.EventAttendee{
			Email:       attendee.Email,
			DisplayName: attendee.DisplayName,
		})
	}

	var calendarReminders = &calendar.EventReminders{}
	for _, reminder := range event.Reminders {
		if reminder.Actions == models.ReminderActionDisplay {
			calendarReminders.ForceSendFields = []string{"UseDefault"}
			calendarReminders.Overrides = append(calendarReminders.Overrides, &calendar.EventReminder{
				Method:  "popup",
				Minutes: int64(event.StartTime.Sub(reminder.Trigger.PointInTime).Minutes()),
			})
		}
	}

	_, err := retry(ctx, func() (*calendar.Event, error) {
		g.RateLimiter.Take()
		return g.Client.Events.Update(g.CalendarId, event.ID, &calendar.Event{
			Summary:            event.Title,
			Description:        event.Description,
			Location:           event.Location,
			Start:              timeToEventDateTime(event.AllDay, event.StartTime),
			End:                timeToEventDateTime(event.AllDay, event.EndTime),
			ExtendedProperties: extProperties,
			Attendees:          calendarAttendees,
			Reminders:          calendarReminders,
		}).Context(ctx).SendUpdates("none").Do()
	})
	if isNotFound(err) {
		return errors.New("already deleted")
	} else if err != nil {
		return err
	}
	return nil

}

func (g *GCalClient) DeleteEvent(ctx context.Context, event models.Event) error {
	_, err := retry(ctx, func() (*calendar.Event, error) {
		g.RateLimiter.Take()
		err := g.Client.Events.Delete(g.CalendarId, event.ID).Context(ctx).SendUpdates("none").Do()
		return nil, err
	})
	if isNotFound(err) {
		g.logger.Debug("Event is already deleted.", "method", "DeleteEvent", "title", event.ShortTitle(), "time", event.StartTime.String())
		return nil
	} else if err != nil {
		return err
	}
	return nil
}

// loadPages recursively loads all pages starting with the given nextPageToken.
// All resulting events are appended to the 'events' slice pointer.
func (g *GCalClient) loadPages(listCall *calendar.EventsListCall, events *[]*calendar.Event, nextPageToken string) error {
	g.RateLimiter.Take()
	pageEvents, err := listCall.PageToken(nextPageToken).Do()
	if err != nil {
		return fmt.Errorf("loadPages failed: %w", err)
	}
	*events = append(*events, pageEvents.Items...)

	if pageEvents.NextPageToken == "" {
		return nil
	}
	return g.loadPages(listCall, events, pageEvents.NextPageToken)
}

// GetCalendarID calculates a unique ID for this adapter based on the current calendar.
// This is used to distinguish between adapters in order to not overwrite or delete events
// which are maintained by different adapters.
// A simple use-case for this is if you have multiple google calendars as source adapters configured.
func (g *GCalClient) GetCalendarID() string {
	var id []byte

	sum := sha1.Sum([]byte(g.CalendarId))
	id = append(id, sum[:]...)
	return base64.URLEncoding.EncodeToString(id)
}
