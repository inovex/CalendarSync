package transformation

import (
	"fmt"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"
)

// KeepMeetingLink allows to keep the meeting link of an event.
type KeepMeetingLink struct{}

func (t *KeepMeetingLink) Name() string {
	return "KeepMeetingLink"
}

func (t *KeepMeetingLink) Transform(source models.Event, sink models.Event) (models.Event, error) {
	if len(source.MeetingLink) > 0 {
		sink.Description = fmt.Sprintf("original meeting link: %s\n\n############\n%s", source.MeetingLink, sink.Description)
	}
	return sink, nil
}
