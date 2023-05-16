package transformation

import (
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"
)

// PrefixTitle allows to replace the title of an event.
type PrefixTitle struct {
	Prefix string
}

func (t *PrefixTitle) Name() string {
	return "PrefixTitle"
}

func (t *PrefixTitle) Transform(_ models.Event, sink models.Event) (models.Event, error) {
	sink.Title = t.Prefix + sink.Title
	return sink, nil
}
