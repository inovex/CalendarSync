package apple

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/emersion/go-ical"

	"github.com/inovex/CalendarSync/internal/models"
)

// ACalClient implements the AppleCalendarClient interface
type ACalClient struct {
	Username    string
	AppPassword string
	CalendarID  string
	httpClient  *http.Client
}

type CalDAVPropfind struct {
	XMLName xml.Name   `xml:"DAV: propfind"`
	Prop    CalDAVProp `xml:"DAV: prop"`
}

type CalDAVProp struct {
	GetETag      xml.Name `xml:"DAV: getetag"`
	CalendarData xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
}

type CalDAVMultistatus struct {
	XMLName   xml.Name         `xml:"DAV: multistatus"`
	Responses []CalDAVResponse `xml:"DAV: response"`
}

type CalDAVResponse struct {
	Href     string         `xml:"DAV: href"`
	Propstat CalDAVPropstat `xml:"DAV: propstat"`
}

type CalDAVPropstat struct {
	Prop   CalDAVPropData `xml:"DAV: prop"`
	Status string         `xml:"DAV: status"`
}

type CalDAVPropData struct {
	ETag         string `xml:"DAV: getetag"`
	CalendarData string `xml:"urn:ietf:params:xml:ns:caldav calendar-data"`
}

type CalDAVCompFilter struct {
	Name       string            `xml:"name,attr"`
	TimeRange  *CalDAVTimeRange  `xml:"urn:ietf:params:xml:ns:caldav time-range,omitempty"`
	CompFilter *CalDAVCompFilter `xml:"urn:ietf:params:xml:ns:caldav comp-filter,omitempty"`
}

type CalDAVTimeRange struct {
	Start string `xml:"start,attr"`
	End   string `xml:"end,attr"`
}

type CalDAVResourceType struct {
	Collection xml.Name `xml:"DAV: collection"`
	Calendar   xml.Name `xml:"urn:ietf:params:xml:ns:caldav calendar"`
}

type CalDAVComponentSet struct {
	Components []CalDAVComponent `xml:"urn:ietf:params:xml:ns:caldav comp"`
}

type CalDAVComponent struct {
	Name string `xml:"name,attr"`
}

type CalDAVCalendarList struct {
	XMLName   xml.Name                 `xml:"DAV: multistatus"`
	Responses []CalDAVCalendarResponse `xml:"DAV: response"`
}

type CalDAVCalendarResponse struct {
	Href     string                 `xml:"DAV: href"`
	Propstat CalDAVCalendarPropstat `xml:"DAV: propstat"`
}

type CalDAVCalendarPropstat struct {
	Prop   CalDAVCalendarProp `xml:"DAV: prop"`
	Status string             `xml:"DAV: status"`
}

type CalDAVCalendarProp struct {
	DisplayName  string             `xml:"DAV: displayname"`
	ResourceType CalDAVResourceType `xml:"DAV: resourcetype"`
	ComponentSet CalDAVComponentSet `xml:"urn:ietf:params:xml:ns:caldav supported-calendar-component-set"`
}

type CalDAVCalendarQuery struct {
	XMLName xml.Name                  `xml:"urn:ietf:params:xml:ns:caldav calendar-query"`
	Prop    CalDAVProp                `xml:"DAV: prop"`
	Filter  CalDAVCalendarQueryFilter `xml:"urn:ietf:params:xml:ns:caldav filter"`
}

type CalDAVCalendarQueryFilter struct {
	CompFilter CalDAVCompFilter `xml:"urn:ietf:params:xml:ns:caldav comp-filter"`
}

func (c *ACalClient) initHttpClient() {
	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
}

func (c *ACalClient) makeRequest(ctx context.Context, method, path string, body []byte, headers map[string]string) (*http.Response, error) {
	c.initHttpClient()

	url := fmt.Sprintf("%s%s", baseUrl, path)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	// Set basic auth
	req.SetBasicAuth(c.Username, c.AppPassword)

	// Set default headers
	req.Header.Set("User-Agent", "CalendarSync/1.0")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")

	// Set additional headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		switch resp.StatusCode {
		case 401:
			return nil, fmt.Errorf("authentication failed - check Apple ID and app-specific password")
		case 404:
			return nil, fmt.Errorf("calendar not found - check calendar ID '%s'", c.CalendarID)
		case 403:
			return nil, fmt.Errorf("access denied - check calendar permissions")
		default:
			return nil, fmt.Errorf("CalDAV request failed with status %d: %s", resp.StatusCode, string(body))
		}
	}

	return resp, nil
}

func (c *ACalClient) DiscoverPrincipal(ctx context.Context) (string, error) {
	propfindXML := `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:">
  <D:prop>
    <D:current-user-principal />
  </D:prop>
</D:propfind>`

	resp, err := c.makeRequest(ctx, "PROPFIND", "/.well-known/caldav", []byte(propfindXML), map[string]string{
		"Depth": "0",
	})
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse to extract principal ID from href like "/ID/principal/"
	var multistatus CalDAVMultistatus
	if err := xml.Unmarshal(responseBody, &multistatus); err != nil {
		return "", fmt.Errorf("failed to parse principal response: %w", err)
	}

	if len(multistatus.Responses) > 0 {
		href := multistatus.Responses[0].Href
		// Extract principal ID from "/ID/principal/"
		parts := strings.Split(strings.Trim(href, "/"), "/")
		if len(parts) >= 1 {
			return parts[0], nil
		}
	}

	return "", fmt.Errorf("could not discover principal ID")
}

func (c *ACalClient) ResolveCalendarID(ctx context.Context, friendlyName string) (string, string, error) {
	principalID, err := c.DiscoverPrincipal(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to discover principal: %w", err)
	}

	calendars, err := c.discoverCalendarsWithPrincipal(ctx, principalID)
	if err != nil {
		return "", "", fmt.Errorf("failed to discover calendars: %w", err)
	}

	if len(calendars) == 0 {
		return "", "", fmt.Errorf("no calendars found")
	}

	// Check if it's already a UUID
	if len(friendlyName) > 30 && strings.Contains(friendlyName, "-") {
		for _, calendarID := range calendars {
			if calendarID == friendlyName {
				return principalID, calendarID, nil
			}
		}
	}

	// Look up by friendly name (case-insensitive exact match)
	calendarID, exists := calendars[strings.ToLower(friendlyName)]
	if exists {
		return principalID, calendarID, nil
	}

	// If exact match fails, try partial matching
	var matches []string
	var matchedCalendarID string

	for name, id := range calendars {
		if strings.Contains(strings.ToLower(name), strings.ToLower(friendlyName)) {
			matches = append(matches, name)
			matchedCalendarID = id
		}
	}

	// If exactly one partial match, use it
	if len(matches) == 1 {
		log.Printf("Resolved '%s' to calendar: '%s' -> %s (partial match)", friendlyName, matches[0], matchedCalendarID)
		return principalID, matchedCalendarID, nil
	}

	// If multiple matches, that's ambiguous
	if len(matches) > 1 {
		return "", "", fmt.Errorf("calendar name '%s' is ambiguous. Multiple matches found: %v", friendlyName, matches)
	}

	// No matches found
	availableCalendars := make([]string, 0, len(calendars))
	for name := range calendars {
		availableCalendars = append(availableCalendars, name)
	}

	return "", "", fmt.Errorf("calendar '%s' not found. Available calendars: %v", friendlyName, availableCalendars)
}

func (c *ACalClient) DiscoverCalendars(ctx context.Context) ([]string, error) {
	principalID, err := c.DiscoverPrincipal(ctx)
	if err != nil {
		return nil, err
	}

	calendars, err := c.discoverCalendarsWithPrincipal(ctx, principalID)
	if err != nil {
		return nil, err
	}

	var calendarNames []string
	for name := range calendars {
		calendarNames = append(calendarNames, name)
	}

	return calendarNames, nil
}

func (c *ACalClient) ListEvents(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	principalID, resolvedCalendarID, err := c.ResolveCalendarID(ctx, c.CalendarID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve calendar '%s': %w", c.CalendarID, err)
	}

	// Build CalDAV query
	query := CalDAVCalendarQuery{
		Prop: CalDAVProp{
			GetETag:      xml.Name{},
			CalendarData: xml.Name{},
		},
		Filter: CalDAVCalendarQueryFilter{
			CompFilter: CalDAVCompFilter{
				Name: "VCALENDAR",
				CompFilter: &CalDAVCompFilter{
					Name: "VEVENT",
					TimeRange: &CalDAVTimeRange{
						Start: start.UTC().Format(timeFormat),
						End:   end.UTC().Format(timeFormat),
					},
				},
			},
		},
	}

	calendarPath := fmt.Sprintf("/%s/calendars/%s/", principalID, resolvedCalendarID)

	queryXML, err := xml.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CalDAV query: %w", err)
	}

	resp, err := c.makeRequest(ctx, "REPORT", calendarPath, queryXML, map[string]string{
		"Depth": "1",
	})
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var multistatus CalDAVMultistatus
	if err := xml.Unmarshal(responseBody, &multistatus); err != nil {
		return nil, fmt.Errorf("failed to parse CalDAV response: %w", err)
	}

	var events []models.Event
	for _, response := range multistatus.Responses {
		if response.Propstat.Prop.CalendarData != "" {
			event, err := c.parseICalendarData(response.Propstat.Prop.CalendarData, response.Href)
			if err != nil {
				log.Errorf("failed to parse calendar data: %v", err)
				continue
			}
			if event != nil {
				events = append(events, *event)
			}
		}
	}

	return events, nil
}

func (c *ACalClient) parseICalendarData(icalData, href string) (*models.Event, error) {
	decoder := ical.NewDecoder(strings.NewReader(icalData))
	cal, err := decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to parse iCalendar data: %w", err)
	}

	for _, child := range cal.Children {
		if child.Name == ical.CompEvent {
			return c.convertVEventToEvent(child, href), nil
		}
	}

	return nil, fmt.Errorf("no VEVENT found in calendar data")
}

func (c *ACalClient) convertVEventToEvent(vevent *ical.Component, href string) *models.Event {
	event := &models.Event{
		Metadata: models.NewEventMetadata(href, href, c.GetCalendarHash()),
	}

	// Extract basic properties
	if summary, _ := vevent.Props.Text(ical.PropSummary); summary != "" {
		event.Title = summary
	}

	if description, _ := vevent.Props.Text(ical.PropDescription); description != "" {
		event.Description = description
	}

	if location, _ := vevent.Props.Text(ical.PropLocation); location != "" {
		event.Location = location
	}

	if uid, _ := vevent.Props.Text(ical.PropUID); uid != "" {
		event.ICalUID = uid
		event.ID = uid
	}

	// Extract datetime properties
	if startTime, err := vevent.Props.DateTime(ical.PropDateTimeStart, time.Local); err == nil && !startTime.IsZero() {
		event.StartTime = startTime
	}

	if endTime, err := vevent.Props.DateTime(ical.PropDateTimeEnd, time.Local); err == nil && !endTime.IsZero() {
		event.EndTime = endTime
	}

	// Check if it's an all-day event
	if startProp := vevent.Props.Get(ical.PropDateTimeStart); startProp != nil {
		event.AllDay = startProp.ValueType() == ical.ValueDate
	}

	event.Accepted = true
	return event
}

func (c *ACalClient) CreateEvent(ctx context.Context, event models.Event) error {
	icalData := c.eventToICalendar(event)

	// Generate unique filename
	filename := fmt.Sprintf("%s.ics", event.Metadata.SyncID)
	eventPath := fmt.Sprintf("/%s/calendars/%s/%s/%s", c.Username, c.Username, c.CalendarID, filename)

	_, err := c.makeRequest(ctx, "PUT", eventPath, []byte(icalData), map[string]string{
		"Content-Type": "text/calendar; charset=utf-8",
	})

	return err
}

func (c *ACalClient) UpdateEvent(ctx context.Context, event models.Event) error {
	// For updates, we use the same PUT method with the existing resource path
	return c.CreateEvent(ctx, event)
}

func (c *ACalClient) DeleteEvent(ctx context.Context, event models.Event) error {
	filename := fmt.Sprintf("%s.ics", event.Metadata.SyncID)
	eventPath := fmt.Sprintf("/%s/calendars/%s/%s/%s", c.Username, c.Username, c.CalendarID, filename)

	_, err := c.makeRequest(ctx, "DELETE", eventPath, nil, nil)
	return err
}

func (c *ACalClient) discoverCalendarsWithPrincipal(ctx context.Context, principalID string) (map[string]string, error) {
	propfindXML := `<?xml version="1.0" encoding="utf-8"?>
<D:propfind xmlns:D="DAV:" xmlns:C="urn:ietf:params:xml:ns:caldav">
  <D:prop>
    <D:displayname />
    <D:resourcetype />
    <C:supported-calendar-component-set />
  </D:prop>
</D:propfind>`

	calendarsPath := fmt.Sprintf("/%s/calendars/", principalID)
	resp, err := c.makeRequest(ctx, "PROPFIND", calendarsPath, []byte(propfindXML), map[string]string{
		"Depth": "1",
	})
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var calendarList CalDAVCalendarList
	if err := xml.Unmarshal(responseBody, &calendarList); err != nil {
		return nil, fmt.Errorf("failed to parse calendar list: %w", err)
	}

	calendars := make(map[string]string)

	for _, response := range calendarList.Responses {
		// Check if this is actually a calendar (has calendar resource type)
		isCalendar := false
		supportsEvents := false

		// Check resource type
		if response.Propstat.Prop.ResourceType.Calendar.Local != "" {
			isCalendar = true
		}

		// Check if it supports VEVENT
		for _, comp := range response.Propstat.Prop.ComponentSet.Components {
			if comp.Name == "VEVENT" {
				supportsEvents = true
				break
			}
		}

		if isCalendar && supportsEvents && response.Propstat.Prop.DisplayName != "" {
			// Extract calendar ID from href like "/ID/calendars/UUID/"
			parts := strings.Split(strings.Trim(response.Href, "/"), "/")
			if len(parts) >= 3 {
				calendarID := parts[2]
				displayName := response.Propstat.Prop.DisplayName
				calendars[strings.ToLower(displayName)] = calendarID
			}
		}
	}

	return calendars, nil
}

func (c *ACalClient) eventToICalendar(event models.Event) string {
	// Create calendar
	cal := ical.NewCalendar()
	cal.Props.SetText(ical.PropVersion, "2.0")
	cal.Props.SetText(ical.PropProductID, "-//CalendarSync//CalendarSync 1.0//EN")

	// Create event
	vevent := ical.NewComponent(ical.CompEvent)
	vevent.Props.SetText(ical.PropUID, event.ICalUID)
	vevent.Props.SetDateTime(ical.PropDateTimeStamp, time.Now().UTC())

	// Handle all-day vs timed events
	if event.AllDay {
		vevent.Props.SetDate(ical.PropDateTimeStart, event.StartTime)
		vevent.Props.SetDate(ical.PropDateTimeEnd, event.EndTime)
	} else {
		vevent.Props.SetDateTime(ical.PropDateTimeStart, event.StartTime.UTC())
		vevent.Props.SetDateTime(ical.PropDateTimeEnd, event.EndTime.UTC())
	}

	if event.Title != "" {
		vevent.Props.SetText(ical.PropSummary, event.Title)
	}
	if event.Description != "" {
		vevent.Props.SetText(ical.PropDescription, event.Description)
	}
	if event.Location != "" {
		vevent.Props.SetText(ical.PropLocation, event.Location)
	}

	cal.Children = append(cal.Children, vevent)

	// Encode to string
	var buf strings.Builder
	encoder := ical.NewEncoder(&buf)
	if err := encoder.Encode(cal); err != nil {
		// Fallback to manual format if encoding fails
		return c.eventToICalendarManual(event)
	}

	return buf.String()
}

func (c *ACalClient) eventToICalendarManual(event models.Event) string {
	return fmt.Sprintf(`BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//CalendarSync//CalendarSync 1.0//EN
BEGIN:VEVENT
UID:%s
DTSTAMP:%s
DTSTART:%s
DTEND:%s
SUMMARY:%s
DESCRIPTION:%s
LOCATION:%s
END:VEVENT
END:VCALENDAR`,
		event.ICalUID,
		time.Now().UTC().Format(timeFormat),
		event.StartTime.UTC().Format(timeFormat),
		event.EndTime.UTC().Format(timeFormat),
		event.Title,
		event.Description,
		event.Location,
	)
}

func (c *ACalClient) GetCalendarHash() string {
	var id []byte
	components := []string{c.Username, c.CalendarID}

	sum := sha1.Sum([]byte(strings.Join(components, "")))
	id = append(id, sum[:]...)
	return base64.URLEncoding.EncodeToString(id)
}
