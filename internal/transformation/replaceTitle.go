package transformation

import (
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"
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
