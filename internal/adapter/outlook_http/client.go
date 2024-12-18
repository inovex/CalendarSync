package outlook_http

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/charmbracelet/log"

	"github.com/inovex/CalendarSync/internal/models"
)

const (
	ExtensionOdataType = "microsoft.graph.openTypeExtension"
	ExtensionName      = "inovex.calendarsync.meta"
)

// OutlookClient implements the OutlookCalendarClient interface
type OutlookClient struct {
	Client     *http.Client
	CalendarID string
}

func (o *OutlookClient) ListEvents(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
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

	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	var eventList EventList
	err = json.Unmarshal(body, &eventList)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal response: %w", err)
	}

	nextLink := eventList.NextLink
	for nextLink != "" {
		resp, err := o.Client.Get(nextLink)
		if err != nil {
			return nil, err
		}

		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()

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

// CreateEvent creates an event in the outlook sink
// When an event is sent, the server sends invitations to all the attendees.
// https://learn.microsoft.com/en-us/graph/api/user-post-events?view=graph-rest-1.0&tabs=http
func (o *OutlookClient) CreateEvent(ctx context.Context, event models.Event) error {
	outlookEvent := o.eventToOutlookEvent(event)
	by, err := json.Marshal(outlookEvent)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseUrl+"/me/calendars/"+o.CalendarID+"/events", bytes.NewBuffer(by))
	if err != nil {
		return err
	}
	req.Header.Set("Content-type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// TODO: we can maybe do this better
	// the error messages are maybe standardized
	if resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		log.Debugf("Create Operation Response Body: %s", string(body))
		return fmt.Errorf("status code at event creation was not 201, response: %v", string(body))
	}
	return nil
}

// UpdateEvent updates the event when used as a sink
func (o *OutlookClient) UpdateEvent(ctx context.Context, event models.Event) error {
	// https://learn.microsoft.com/en-us/graph/api/event-update?view=graph-rest-1.0&tabs=http
	// Normally in a patch operation we would update only the fields which changed
	// but just update everything for simplicity
	outlookEvent := o.eventToOutlookEvent(event)
	by, err := json.Marshal(outlookEvent)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, baseUrl+"/me/calendars/"+o.CalendarID+"/events/"+event.ID, bytes.NewBuffer(by))
	if err != nil {
		return err
	}
	req.Header.Set("Content-type", "application/json")
	resp, err := o.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code at event update was not 200, response: %v", string(body))
	}
	return nil
}

func (o *OutlookClient) DeleteEvent(ctx context.Context, event models.Event) error {
	// https://learn.microsoft.com/en-us/graph/api/event-delete?view=graph-rest-1.0&tabs=http
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, baseUrl+"/me/calendars/"+o.CalendarID+"/events/"+event.ID, nil)
	if err != nil {
		return err
	}
	_, err = o.Client.Do(req)
	if err != nil {
		return err
	}

	return nil
}

func (o OutlookClient) GetCalendarHash() string {
	var id []byte
	sum := sha1.Sum([]byte(o.CalendarID))
	id = append(id, sum[:]...)
	return base64.URLEncoding.EncodeToString(id)
}

// eventToOutlookEvent transforms our internal models.Event to the outlook format
// will get called when using the outlook adapter as a sink
func (o OutlookClient) eventToOutlookEvent(e models.Event) (oe Event) {
	outlookEvent := Event{}
	outlookEvent.Location.Name = e.Location

	outlookEvent.Start.DateTime = e.StartTime.UTC().Format(timeFormat)
	outlookEvent.Start.TimeZone = "UTC"
	outlookEvent.End.DateTime = e.EndTime.UTC().Format(timeFormat)
	outlookEvent.End.TimeZone = "UTC"

	outlookEvent.Subject = e.Title
	outlookEvent.ID = e.ID
	outlookEvent.UID = e.ICalUID

	if len(e.Description) > 0 {
		outlookEvent.Body.Content = e.Description
		outlookEvent.Body.ContentType = "text"
	}

	calendarSyncExtension := &Extensions{
		OdataType:     ExtensionOdataType,
		ExtensionName: ExtensionName,

		Metadata: models.Metadata{
			SyncID:           e.Metadata.SyncID,
			SourceID:         e.Metadata.SourceID,
			OriginalEventUri: e.Metadata.OriginalEventUri,
		},
	}
	outlookEvent.Extensions = append(outlookEvent.Extensions, *calendarSyncExtension)

	for _, att := range e.Attendees {
		outlookEvent.Attendees = append(outlookEvent.Attendees, Attendee{
			EmailAddress: EmailAddress{
				Address: att.Email,
				Name:    att.DisplayName,
			},
		})
	}

	if e.AllDay {
		outlookEvent.IsAllDay = true
	}

	if len(e.Reminders) != 0 {
		outlookEvent.IsReminderOn = true
		// we currently use the first reminder in the list, this may result in data loss
		outlookEvent.ReminderMinutesBeforeStart = int(e.StartTime.Sub(e.Reminders[0].Trigger.PointInTime).Minutes())
	}
	return outlookEvent
}

// outlookEventToEvent transforms an outlook event to our form of event representation
// gets called when used as a sink and as a source
func (o OutlookClient) outlookEventToEvent(oe Event, adapterSourceID string) (e models.Event, err error) {
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
