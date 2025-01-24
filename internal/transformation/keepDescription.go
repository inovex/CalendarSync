package transformation

import (
	"strings"
	"sync"

	"github.com/aquilax/truncate"
	"github.com/inovex/CalendarSync/internal/models"
	"github.com/microcosm-cc/bluemonday"
)

// KeepDescription allows to keep the description of an event.
type KeepDescription struct {
	policy     *bluemonday.Policy
	initPolicy sync.Once
}

func (t *KeepDescription) Name() string {
	return "KeepDescription"
}

func (t *KeepDescription) Transform(source models.Event, sink models.Event) (models.Event, error) {
	t.initPolicy.Do(func() {
		t.policy = bluemonday.UGCPolicy()
	})

	// need to remove microsoft html overhead (the description in outlook contains a lot of '\r\n's)
	description := strings.ReplaceAll(source.Description, "\r\n", "")
	sanitizedDescription := t.policy.Sanitize(description)
	sanitizedDescription2 := strings.TrimSpace(sanitizedDescription)

	// Since the description cannot exceed a specified amount in some sinks (e.g. google)
	// we're cutting the desc at 4000 chars here.
	sink.Description = truncate.Truncate(sanitizedDescription2, 4000, "...", truncate.PositionEnd)
	return sink, nil
}
