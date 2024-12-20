package ics

import (
	"context"
	"github.com/inovex/CalendarSync/internal/models"
	"time"
)

type ICalendarClient interface {
	ListEvents(ctx context.Context, starttime time.Time, enddtime time.Time) ([]models.Event, error)
}

type CalendarAPI struct {
	icalendarClient ICalendarClient
	calendarUrl     string
}

func (c *CalendarAPI) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	events, err := c.icalendarClient.ListEvents(ctx, start, end)
	if err != nil {
		return nil, err
	}

	//c.logger.Infof("loaded %d events between %s and %s.", len(events), start.Format(time.RFC1123), end.Format(time.RFC1123))

	return events, nil
}

func (c *CalendarAPI) Name() string {
	return "iCalendar"
}

func (c *CalendarAPI) GetCalendarID() string {
	return c.calendarUrl
}

func (c *CalendarAPI) Initialize(ctx context.Context, openBrowser bool, config map[string]interface{}) error {
	c.calendarUrl = config["url"].(string)

	c.icalendarClient = &ICalClient{url: c.calendarUrl}
	return nil
}
