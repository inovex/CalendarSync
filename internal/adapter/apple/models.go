package apple

import (
	"github.com/inovex/CalendarSync/internal/models"
)

// CalDAVEvent represents an event as stored in CalDAV format
type CalDAVEvent struct {
	ID          string
	ICalUID     string
	ETag        string
	Href        string
	Summary     string
	Description string
	Location    string
	StartTime   string
	EndTime     string
	AllDay      bool
}

// CalDAVExtensions extensions for CalendarSync metadata in CalDAV
type CalDAVExtensions struct {
	ExtensionName string `xml:"extensionName"`
	// Embed CalendarSync metadata
	models.Metadata
}
