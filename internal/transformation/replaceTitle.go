package transformation

import (
	"github.com/inovex/CalendarSync/internal/models"
)

// ReplaceTitle allows to replace the title of an event.
type ReplaceTitle struct {
	NewTitle string
}

func (t *ReplaceTitle) Name() string {
	return "ReplaceTitle"
}

func (t *ReplaceTitle) Transform(_ models.Event, sink models.Event) (models.Event, error) {
	sink.Title = t.NewTitle
	return sink, nil
}
