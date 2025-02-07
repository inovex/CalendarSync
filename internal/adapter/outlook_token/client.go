package outlook_token

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/inovex/CalendarSync/internal/models"
)

const (
	ExtensionOdataType = "microsoft.graph.openTypeExtension"
	ExtensionName      = "inovex.calendarsync.meta"
)

// ROOutlookClient implements the ROOutlookCalendarClient interface
type ROOutlookClient struct {
	AccessToken string
	CalendarID  string

	Client *http.Client
}

func (o *ROOutlookClient) ListEvents(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	startDate := start.Format(timeFormat)
	endDate := end.Format(timeFormat)

	// Query can't simply be encoded with the url package for example, microsoft also uses its own encoding here.
	// Otherwise this always ends in a 500 return code, see also https://stackoverflow.com/a/62770941
	query := "?startDateTime=" + startDate + "&endDateTime=" + endDate + "&$expand=extensions($filter=Id%20eq%20'inovex.calendarsync.meta')"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseUrl+"/me/calendars/"+o.CalendarID+"/CalendarView"+query, nil)
	if err != nil {
		return nil, err
	}

	// Get all the events in UTC timezone
	// when we retrieve them from other adapters they will also be in UTC
	req.Header.Add("Prefer", "outlook.timezone=\"UTC\"")
	req.Header.Add("Authorization", "Bearer "+o.AccessToken)

	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var eventList EventList
	err = json.Unmarshal(body, &eventList)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal response: %w", err)
	}

	nextLink := eventList.NextLink
	for nextLink != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, nextLink, nil)
		req.Header.Add("Prefer", "outlook.timezone=\"UTC\"")
		req.Header.Add("Authorization", "Bearer "+o.AccessToken)

		resp, err := o.Client.Do(req)
		if err != nil {
			return nil, err
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var nextList EventList
		err = json.Unmarshal(body, &nextList)
		if err != nil {
			return nil, fmt.Errorf("cannot unmarshal response: %w", err)
		}

		eventList.Events = append(eventList.Events, nextList.Events...)
		nextLink = nextList.NextLink
	}

	var events []models.Event
	for _, evt := range eventList.Events {
		evt, err := o.outlookEventToEvent(evt, o.GetCalendarHash())
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}

	return events, nil
}

func (o ROOutlookClient) GetCalendarHash() string {
	var id []byte
	sum := sha1.Sum([]byte(o.CalendarID))
	id = append(id, sum[:]...)
	return base64.URLEncoding.EncodeToString(id)
}

// outlookEventToEvent transforms an outlook event to our form of event representation
// gets called when used as a sink and as a source
func (o ROOutlookClient) outlookEventToEvent(oe Event, adapterSourceID string) (e models.Event, err error) {
	var bufEvent models.Event

	startTime, err := time.Parse(timeFormat, oe.Start.DateTime)
	if err != nil {
		return bufEvent, fmt.Errorf("failed to parse startTime, skipping event: %s", err)
	}
	endTime, err := time.Parse(timeFormat, oe.End.DateTime)
	if err != nil {
		return bufEvent, fmt.Errorf("failed to parse endTime, skipping event: %s", err)
	}

	var attendees = make([]models.Attendee, 0)

	for _, eventAttendee := range oe.Attendees {
		attendees = append(attendees, models.Attendee{
			Email:       eventAttendee.EmailAddress.Address,
			DisplayName: eventAttendee.EmailAddress.Name,
		})
	}

	var reminders = make([]models.Reminder, 0)

	if oe.IsReminderOn {
		reminders = append(reminders, models.Reminder{
			Actions: models.ReminderActionDisplay,
			Trigger: models.ReminderTrigger{
				PointInTime: startTime.Add(-(time.Minute * time.Duration(oe.ReminderMinutesBeforeStart))),
			},
		})
	}
	var hasEventAccepted bool = true
	if oe.ResponseStatus.Response == "declined" {
		hasEventAccepted = false
	}

	bufEvent = models.Event{
		ICalUID:     oe.UID,
		ID:          oe.ID,
		Title:       oe.Subject,
		Description: oe.Body.Content,
		Location:    oe.Location.Name,
		StartTime:   startTime,
		EndTime:     endTime,
		Metadata:    ensureMetadata(oe, adapterSourceID),
		Attendees:   attendees,
		Reminders:   reminders,
		MeetingLink: oe.OnlineMeetingUrl,
		Accepted:    hasEventAccepted,
	}

	if oe.IsAllDay {
		bufEvent.AllDay = true
	}

	return bufEvent, nil
}

// Adding metadata is a bit more complicated as in the google adapter
// see also: https://learn.microsoft.com/en-us/graph/api/opentypeextension-post-opentypeextension?view=graph-rest-1.0&tabs=http
// Retrieve metadata if possible otherwise regenerate it
func ensureMetadata(event Event, adapterSourceID string) *models.Metadata {
	for _, extension := range event.Extensions {
		if extension.ExtensionName == ExtensionName && (len(extension.SyncID) != 0 && len(extension.SourceID) != 0) {
			return &models.Metadata{
				SyncID:           extension.SyncID,
				OriginalEventUri: extension.OriginalEventUri,
				SourceID:         extension.SourceID,
			}
		}
	}
	return models.NewEventMetadata(event.ID, event.HtmlLink, adapterSourceID)
}
