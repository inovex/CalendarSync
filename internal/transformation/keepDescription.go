package transformation

import (
	"strings"

	"github.com/aquilax/truncate"
	"github.com/microcosm-cc/bluemonday"
	"github.com/inovex/CalendarSync/internal/models"
)

// KeepDescription allows to keep the description of an event.
type KeepDescription struct{}

func (t *KeepDescription) Name() string {
	return "KeepDescription"
}

func (t *KeepDescription) Transform(source models.Event, sink models.Event) (models.Event, error) {
	// need to remove microsoft html overhead
	p := bluemonday.StrictPolicy()
	description := strings.ReplaceAll(source.Description, "\r\n", "")
	sanitizedDescription := p.Sanitize(description)
	sanitizedDescription2 := strings.TrimSpace(sanitizedDescription)

	// Since the description cannot exceed a specified amount in some sinks (e.g. google)
	// we're cutting the desc at 4000 chars here.
	sink.Description = truncate.Truncate(sanitizedDescription2, 4000, "...", truncate.PositionEnd)
	return sink, nil
}
