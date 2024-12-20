package outlook_published

import (
	"context"
	"github.com/charmbracelet/log"
	"github.com/inovex/CalendarSync/internal/models"
	"time"
)

type OutlookPublishedClient interface {
	ListEvents(ctx context.Context, starttime time.Time, enddtime time.Time) ([]models.Event, error)
}

type CalendarAPI struct {
	opb         OutlookPublishedClient
	calendarUrl string
	urlPostData string
	logger      *log.Logger
}

func (c *CalendarAPI) EventsInTimeframe(ctx context.Context, start time.Time, end time.Time) ([]models.Event, error) {
	events, err := c.opb.ListEvents(ctx, start, end)
	if err != nil {
		return nil, err
	}

	c.logger.Infof("loaded %d events between %s and %s.", len(events), start.Format(time.RFC1123), end.Format(time.RFC1123))

	return events, nil
}

func (c *CalendarAPI) Name() string {
	return "outlook_published"
}

func (c *CalendarAPI) GetCalendarID() string {
	return c.calendarUrl
}

func (c *CalendarAPI) Initialize(ctx context.Context, openBrowser bool, config map[string]interface{}) error {
	c.calendarUrl = config["url"].(string)
	c.urlPostData = config["postData"].(string)

	c.opb = &OutlookPubClient{url: c.calendarUrl, urlPostData: c.urlPostData}
	return nil
}

func (c *CalendarAPI) SetLogger(logger *log.Logger) {
	c.logger = logger
}
