package outlook_published

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/inovex/CalendarSync/internal/models"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OutlookPubClient struct {
	url         string
	urlPostData string
}

func (opb *OutlookPubClient) ListEvents(ctx context.Context, starttime time.Time, enddtime time.Time) ([]models.Event, error) {
	var loadedEvents []models.Event
	var filteredEvents []models.Event

	decodedUrlPostData, _ := url.QueryUnescape(opb.urlPostData)

	startTimeFormatted := starttime.Format("2006-01-02T15:04:05.000")
	endTimeFormatted := enddtime.Format("2006-01-02T15:04:05.000")

	decodedUrlPostData = strings.Replace(decodedUrlPostData, "2024-11-01T00:00:00.000", startTimeFormatted, 1)
	decodedUrlPostData = strings.Replace(decodedUrlPostData, "2025-01-05T23:59:59.999", endTimeFormatted, 1)

	// Encode the modified URL post data
	encodedUrlPostData := url.QueryEscape(decodedUrlPostData)
	encodedUrlPostData = strings.Replace(encodedUrlPostData, "+", "%20", -1)

	req, err := http.NewRequestWithContext(ctx, "POST", opb.url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("action", "FindItem")
	req.Header.Add("content-length", "0")
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Add("x-owa-urlpostdata", encodedUrlPostData)

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body) // Read the body from the response.
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch events: %s", resp.Status)
	}

	var response Response
	err = json.Unmarshal([]byte(body), &response)
	if err != nil {
		return nil, err
	}

	for _, event := range response.Body.ResponseMessages.Items[0].RootFolder.Items {
		loadedEvents = append(loadedEvents, opb.calendarEventToEvent(event))
	}

	for _, loadedEvent := range loadedEvents {
		if loadedEvent.StartTime.After(starttime) && loadedEvent.EndTime.Before(enddtime) {
			filteredEvents = append(filteredEvents, loadedEvent)
		}
	}

	return filteredEvents, nil
}

func (opb *OutlookPubClient) calendarEventToEvent(e ItemRootFolder) models.Event {
	var metadata models.Metadata
	metadata = models.Metadata{SyncID: e.ItemId.Id, SourceID: opb.url}

	var dateStart time.Time
	var dateEnd time.Time

	dateStart, err := time.Parse(time.RFC3339, e.Start)
	if err != nil {
		fmt.Printf("error parsing start date: %v\n", err)
	}

	dateEnd, err = time.Parse(time.RFC3339, e.End)
	if err != nil {
		fmt.Printf("error parsing end date: %v\n", err)
	}

	return models.Event{
		ICalUID:     e.ItemId.Id,
		ID:          e.ItemId.Id,
		Title:       e.Subject,
		Description: "",
		Location:    "",
		AllDay:      false,
		StartTime:   dateStart,
		EndTime:     dateEnd,
		Metadata:    &metadata,
		//Attendees:
		//Reminders:
		//MeetingLink:
		Accepted: true,
	}
}
