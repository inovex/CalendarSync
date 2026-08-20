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
	"net/url"
	"time"

	"github.com/charmbracelet/log"

	"github.com/inovex/CalendarSync/internal/models"
)

const (
	ExtensionOdataType = "microsoft.graph.openTypeExtension"
	ExtensionName      = "inovex.calendarsync.meta"
	// This identifier must remain stable so every CalendarSync build recognizes shared-mailbox metadata.
	// Its UUID is UUIDv5(URL, "https://github.com/inovex/CalendarSync/outlook-metadata").
	singleValueMetadataID     = "String {23f8dbef-16e9-5e2c-8cc7-e7f020136a50} Name " + ExtensionName
	openExtensionExpand       = "extensions($filter=Id%20eq%20'inovex.calendarsync.meta')"
	singleValueMetadataExpand = "singleValueExtendedProperties($filter=id%20eq%20'String%20%7B23f8dbef-16e9-5e2c-8cc7-e7f020136a50%7D%20Name%20inovex.calendarsync.meta')"
)

// OutlookClient implements the OutlookCalendarClient interface
type OutlookClient struct {
	Client     *http.Client
	CalendarID string
	User       string
}

func (o OutlookClient) calendarPath() string {
	if o.User == "" {
		return "/me/calendars/" + o.CalendarID
	}
	return "/users/" + url.PathEscape(o.User) + "/calendars/" + o.CalendarID
}

func (o OutlookClient) metadataExpand() string {
	if o.User == "" {
		return openExtensionExpand
	}
	return singleValueMetadataExpand
}

func (o *OutlookClient) ListEvents(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	startDate := start.Format(timeFormat)
	endDate := end.Format(timeFormat)

	// Query can't simply be encoded with the url package for example, microsoft also uses its own encoding here.
	// Otherwise this always ends in a 500 return code, see also https://stackoverflow.com/a/62770941
	query := "?startDateTime=" + startDate + "&endDateTime=" + endDate + "&$expand=" + o.metadataExpand()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseUrl+o.calendarPath()+"/CalendarView"+query, nil)
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

	body, err := readResponseBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code at event listing was not 200, got status code %d, response: %s", resp.StatusCode, string(body))
	}

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

		body, err := readResponseBody(resp)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("status code at paginated event listing was not 200, got status code %d, response: %s", resp.StatusCode, string(body))
		}

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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseUrl+o.calendarPath()+"/events", bytes.NewBuffer(by))
	if err != nil {
		return err
	}
	req.Header.Set("Content-type", "application/json")

	resp, err := o.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

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

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, baseUrl+o.calendarPath()+"/events/"+event.ID, bytes.NewBuffer(by))
	if err != nil {
		return err
	}
	req.Header.Set("Content-type", "application/json")
	resp, err := o.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

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
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, baseUrl+o.calendarPath()+"/events/"+event.ID, nil)
	if err != nil {
		return err
	}
	resp, err := o.Client.Do(req)
	if err != nil {
		return err
	}
	body, err := readResponseBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("status code at event deletion was not 204, got status code %d, response: %s", resp.StatusCode, string(body))
	}

	return nil
}

func readResponseBody(response *http.Response) ([]byte, error) {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		_ = response.Body.Close()
		return nil, err
	}
	if err := response.Body.Close(); err != nil {
		return nil, err
	}
	return body, nil
}

func (o OutlookClient) GetCalendarHash() string {
	var id []byte
	identity := o.CalendarID
	if o.User != "" {
		identity = o.User + "\x00" + o.CalendarID
	}
	sum := sha1.Sum([]byte(identity))
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

	calendarSyncMetadata := models.Metadata{
		SyncID:           e.Metadata.SyncID,
		SourceID:         e.Metadata.SourceID,
		OriginalEventUri: e.Metadata.OriginalEventUri,
	}
	if o.User == "" {
		outlookEvent.Extensions = append(outlookEvent.Extensions, Extensions{
			OdataType:     ExtensionOdataType,
			ExtensionName: ExtensionName,
			Metadata:      calendarSyncMetadata,
		})
	} else {
		metadataValue, _ := json.Marshal(calendarSyncMetadata)
		outlookEvent.SingleValueExtendedProperties = append(outlookEvent.SingleValueExtendedProperties, SingleValueExtendedProperty{
			ID:    singleValueMetadataID,
			Value: string(metadataValue),
		})
	}

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
	var hasEventAccepted = true
	if oe.ResponseStatus.Response == "declined" {
		hasEventAccepted = false
	}
	metadata, err := ensureMetadata(oe, adapterSourceID)
	if err != nil {
		return bufEvent, err
	}

	bufEvent = models.Event{
		ICalUID:     oe.UID,
		ID:          oe.ID,
		Title:       oe.Subject,
		Description: oe.Body.Content,
		Location:    oe.Location.Name,
		StartTime:   startTime,
		EndTime:     endTime,
		Metadata:    metadata,
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
func ensureMetadata(event Event, adapterSourceID string) (*models.Metadata, error) {
	for _, extension := range event.Extensions {
		if extension.ExtensionName == ExtensionName && (len(extension.SyncID) != 0 && len(extension.SourceID) != 0) {
			return &models.Metadata{
				SyncID:           extension.SyncID,
				OriginalEventUri: extension.OriginalEventUri,
				SourceID:         extension.SourceID,
			}, nil
		}
	}
	for _, property := range event.SingleValueExtendedProperties {
		if property.ID != singleValueMetadataID {
			continue
		}
		var metadata models.Metadata
		if err := json.Unmarshal([]byte(property.Value), &metadata); err != nil {
			return nil, fmt.Errorf("cannot decode Outlook event metadata: %w", err)
		}
		if metadata.SyncID == "" || metadata.SourceID == "" {
			return nil, fmt.Errorf("cannot decode Outlook event metadata: SyncID and SourceID must not be empty")
		}
		return &metadata, nil
	}
	return models.NewEventMetadata(event.ID, event.HtmlLink, adapterSourceID), nil
}
