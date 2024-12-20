package ics

import (
	"context"
	"fmt"
	ical "github.com/arran4/golang-ical"
	"github.com/inovex/CalendarSync/internal/models"
	"io"
	"net/http"
	"strings"
	"time"
)

type ICalClient struct {
	url string
}

func (ic *ICalClient) ListEvents(ctx context.Context, starttime time.Time, enddtime time.Time) ([]models.Event, error) {
	var loadedEvents []models.Event

	req, err := http.NewRequestWithContext(ctx, "GET", ic.url, nil)
	if err != nil {
		return nil, err
	}

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch events: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	cal, err := ical.ParseCalendar(strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	//TODO: Implement starttime and endttime filtering
	eventList := cal.Events()
	for _, event := range eventList {
		loadedEvents = append(loadedEvents, calendarEventToEvent(event))
	}

	return loadedEvents, nil
}

func calendarEventToEvent(e *ical.VEvent) models.Event {
	var summary string
	var location string
	var timeZone string
	var accepted bool
	var allDay bool
	var dateStart time.Time
	var dateEnd time.Time

	for _, prop := range e.ComponentBase.Properties {
		if prop.IANAToken == "X-MICROSOFT-CDO-ALLDAYEVENT" {
			value := prop.Value
			if value == "TRUE" {
				allDay = true
			} else {
				allDay = false
			}
			break
		}
	}

	summary = e.GetProperty(ical.ComponentPropertySummary).Value
	fmt.Println(summary)

	location = e.GetProperty(ical.ComponentPropertyLocation).Value

	status := e.GetProperty(ical.ComponentPropertyStatus).Value
	if status == "CONFIRMED" {
		accepted = true
	} else {
		accepted = false
	}

	layout := "20060102T150405-0700"

	var rawTimeZone string
	if !allDay {
		rawTimeZone = e.GetProperty(ical.ComponentPropertyDtStart).ICalParameters["TZID"][0]
	} else {
		rawTimeZone = ""
	}

	switch rawTimeZone {
	case "Central America Standard Time":
		timeZone = "-0600"
	case "South America Standard Time":
		timeZone = "-0300"
	default:
		timeZone = ""
	}

	renderedStartDate := e.GetProperty(ical.ComponentPropertyDtStart).Value + timeZone
	dateStart, _ = time.Parse(layout, renderedStartDate)

	renderedEndDate := e.GetProperty(ical.ComponentPropertyDtEnd).Value + timeZone
	dateEnd, _ = time.Parse(layout, renderedEndDate)

	return models.Event{
		ICalUID:     e.Id(),
		ID:          e.Id(),
		Title:       summary,
		Description: "",
		Location:    location,
		AllDay:      allDay,
		StartTime:   dateStart,
		EndTime:     dateEnd,
		//Metadata:
		//Attendees:
		//Reminders:
		//MeetingLink:
		Accepted: accepted,
	}
}
