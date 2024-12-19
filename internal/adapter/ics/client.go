package ics

import (
	"context"
	"github.com/inovex/CalendarSync/internal/models"
	"time"
)

type ICalClient struct {
	url string
}

func (ic *ICalClient) ListEvents(ctx context.Context, starttime time.Time, enddtime time.Time) ([]models.Event, error) {
	return nil, nil
}
