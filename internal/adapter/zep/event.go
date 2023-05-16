package zep

import (
	"fmt"
	"time"
)

const (
	DateFormat = "02.01.2006"
)

// Event is a simplified representation for an event which has been entered in ZEP.
type Event struct {
	ID          string
	Start       time.Time
	End         time.Time
	Summary     string
	Description string
	Category    string
	Etag        string
}

func (a Event) String() string {
	return fmt.Sprintf("%s | %s | %s | %s | %s | %s", a.ID, a.Start.Format(DateFormat), a.End.Format(DateFormat), a.Category, a.Summary, a.Description)
}
