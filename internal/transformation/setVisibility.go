package transformation

import (
	"github.com/inovex/CalendarSync/internal/models"
)

// SetVisibility allows to set the visibility of an event.
// Supported values: "default", "public", "private", "confidential"
type SetVisibility struct {
	Visibility string
}

func (t *SetVisibility) Name() string {
	return "SetVisibility"
}

func (t *SetVisibility) Transform(_ models.Event, sink models.Event) (models.Event, error) {
	// Validate visibility value
	validVisibilities := map[string]bool{
		"default":      true,
		"public":       true,
		"private":      true,
		"confidential": true,
	}
	
	if t.Visibility != "" && validVisibilities[t.Visibility] {
		sink.Visibility = t.Visibility
	}
	return sink, nil
}



