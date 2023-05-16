package transformation

import (
	"github.com/inovex/CalendarSync/internal/models"
)

// KeepLocation allows to keep the location of an event.
type KeepLocation struct{}

func (t *KeepLocation) Name() string {
	return "KeepLocation"
}

func (t *KeepLocation) Transform(source models.Event, sink models.Event) (models.Event, error) {
	sink.Location = source.Location
	return sink, nil
}
