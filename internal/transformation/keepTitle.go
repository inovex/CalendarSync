package transformation

import (
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"
)

// KeepTitle allows to keep the title of an event.
type KeepTitle struct{}

func (t *KeepTitle) Name() string {
	return "KeepTitle"
}

func (t *KeepTitle) Transform(source models.Event, sink models.Event) (models.Event, error) {
	sink.Title = source.Title
	return sink, nil
}
