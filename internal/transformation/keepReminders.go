package transformation

import (
	"github.com/inovex/CalendarSync/internal/models"
)

// KeepReminders allows to keep the reminders of an event.
type KeepReminders struct{}

func (t *KeepReminders) Name() string {
	return "KeepReminders"
}

func (t *KeepReminders) Transform(source models.Event, sink models.Event) (models.Event, error) {
	sink.Reminders = source.Reminders
	return sink, nil
}
