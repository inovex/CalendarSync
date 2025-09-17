package transformation

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/inovex/CalendarSync/internal/models"
)

// KeepAttendes allows to keep the attendees of an event.
// Actually to be safe that no email is going anywhere, we're using dummy addresses here. still RFC5322 compliant but updates are going to /dev/null
// Creating a copy of an event with the original email addresses is risky, so this transformer allows you configure:
//   - UseEmailAsDisplayName to populate the email address as attendee display name in the sink, so you're seeing who is attending
type KeepAttendees struct {
	UseEmailAsDisplayName bool
}

func (t *KeepAttendees) Name() string {
	return "KeepAttendees"
}

func (t *KeepAttendees) Transform(source models.Event, sink models.Event) (models.Event, error) {
	var sinkAttendees models.Attendees
	for _, sourceAttendee := range source.Attendees {
		var displayName = sourceAttendee.DisplayName
		if t.UseEmailAsDisplayName {
			displayName = sourceAttendee.Email
		}

		var email = sourceAttendee.Email
		emailTransformed := fmt.Sprintf("%s@localhost", fmt.Sprint(strings.ReplaceAll(email, "@", "_")))

		if _, err := mail.ParseAddress(emailTransformed); err != nil {
			return models.Event{}, fmt.Errorf("no valid email address %s: %w", emailTransformed, err)
		}

		sinkAttendees = append(sinkAttendees, models.Attendee{
			DisplayName: displayName,
			Email:       emailTransformed,
		})
	}
	sink.Attendees = sinkAttendees
	return sink, nil
}
