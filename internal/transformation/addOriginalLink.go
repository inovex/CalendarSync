package transformation

import (
	"fmt"
	"github.com/inovex/CalendarSync/internal/models"
)

type AddOriginalLink struct{}

func (t *AddOriginalLink) Name() string {
	return "AddOriginalLink"
}

func (t *AddOriginalLink) Transform(source models.Event, sink models.Event) (models.Event, error) {
	if len(source.HTMLLink) > 0 {
		sink.Description = fmt.Sprintf("original event link: %s\n\n############\n%s", source.HTMLLink, sink.Description)
	}
	return sink, nil
}
