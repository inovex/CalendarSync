package models

import (
	"sort"
	"time"

	"github.com/charmbracelet/log"
)

// Event describes a calendar event which can be processed in a Controller via Transformer funcs.
//
// TODO: Is Event an interface or a concrete type?
// @ljarosch: 	I think for the time being, it is fine going with a struct.
//
//	But in the future it might be very handy to create different events
//	(maybe because the source provides more/less capabilities).
type Event struct {
	ICalUID     string // RFC5545 iCal UID which is valid across calendaring-systems
	ID          string // a unique ID which is calculated from source event-data. Provides a common ID between original and synced event(s)
	Title       string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	AllDay      bool
	Metadata    *Metadata
	Attendees   Attendees
	Reminders   Reminders
	MeetingLink string
	Accepted    bool
}

type Reminders []Reminder

func (r Reminders) Len() int {
	return len(r)
}

func (r Reminders) Less(i, j int) bool {
	return r[i].Trigger.PointInTime.Before(r[j].Trigger.PointInTime)
}

func (r Reminders) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

type Reminder struct {
	Actions ReminderActions
	Trigger ReminderTrigger
}

func (r *Reminder) Equal(value Reminder) bool {
	return r.Trigger.PointInTime.Equal(value.Trigger.PointInTime)
}

const (
	ReminderActionDisplay ReminderActions = iota
	// TODO: Maybe later
	//	ReminderActionEmail
	//	ReminderActionAudio
)

type ReminderActions int

type ReminderTrigger struct {
	PointInTime time.Time
	// TODO: Add if needed
	//	RepeatCount int
}

// NewSyncEvent derives a new Event from a given source Event.
// The derived event is as bare as possible, it only contains required metadata and the event times.
// It can be aggregated by Transformers with additional data if desired.
func NewSyncEvent(origin Event) Event {
	return Event{
		ICalUID:   origin.ICalUID,
		ID:        origin.ID,
		StartTime: origin.StartTime,
		EndTime:   origin.EndTime,
		AllDay:    origin.AllDay,
		Title:     "CalendarSync Event",
		Metadata:  origin.Metadata,
	}
}

// SyncID returns the event-unique ID, RFC5545 compliant.
// Every event has its own ID and additionally an UID.
// The difference is that, while the UID is the same for every event-series event, the ID differs.
// Additionally, the ID is platform-specific (e.g. Google Calendar).
func (e *Event) SyncID() string {
	return e.Metadata.SyncID
}

// ShortTitle returns the title with a capped max length of maxLen.
// If the title is too long, the string is cut to maxLen and '...' is added
func (e *Event) ShortTitle() string {
	const maxLen = 20

	if len(e.Title) > maxLen {
		return e.Title[:maxLen-1] + "..."
	}
	return e.Title
}

func (e *Event) Overwrite(source Event) Event {
	e.Title = source.Title
	e.Description = source.Description
	e.StartTime = source.StartTime
	e.EndTime = source.EndTime
	e.AllDay = source.AllDay
	e.Metadata = source.Metadata
	e.Attendees = source.Attendees
	e.Location = source.Location
	e.Reminders = source.Reminders
	e.MeetingLink = source.MeetingLink

	return *e
}

// This implementation evalutes the differences after event transformation rather than comparing the content versions
func IsSameEvent(a, b Event) bool {

	// TODO can be done better
	if a.Title != b.Title {
		log.Debugf("Title of Source Event %s at %s changed, Sink Title is %s", a.Title, a.StartTime, b.Title)
		return false
	}

	if a.Description != b.Description {
		log.Debugf("Description of Source Event %s at %s changed", a.Title, a.StartTime)
		log.Debugf("Description in Source: '%s'", a.Description)
		log.Debugf("Description in Sink: '%s'", b.Description)
		return false
	}

	if a.AllDay != b.AllDay {
		log.Debugf("AllDay of Source Event %s at %s changed", a.Title, a.StartTime)
		return false
	}

	if a.AllDay && b.AllDay {
		// only compare dates
		if a.StartTime.Year() != b.StartTime.Year() || a.StartTime.YearDay() != b.StartTime.YearDay() {
			log.Debugf("StartTime of all-day event %s changed, sourceTime: %s, sinkTime: %s", a.Title, a.StartTime.Format(time.DateOnly), b.StartTime.Format(time.DateOnly))
			return false
		}
		if a.EndTime.Year() != b.EndTime.Year() || a.EndTime.YearDay() != b.EndTime.YearDay() {
			log.Debugf("EndTime of all-day event %s changed, sourceTime: %s, sinkTime: %s", a.Title, a.EndTime.Format(time.DateOnly), b.EndTime.Format(time.DateOnly))
			return false
		}
	} else {
		if !a.StartTime.Equal(b.StartTime) {
			log.Debugf("StartTime of Source Event %s changed, sourceTime: %s, sinkTime: %s ", a.Title, a.StartTime, b.StartTime)
			return false
		}
		if !a.EndTime.Equal(b.EndTime) {
			log.Debugf("EndTime of Source Event %s changed, sourceTime: %s, sinkTime: %s ", a.Title, a.StartTime, b.StartTime)
			return false
		}
	}

	if a.AllDay != b.AllDay {
		log.Debugf("AllDay State of Source Event %s at %s changed", a.Title, a.StartTime)
		return false
	}

	if a.Location != b.Location {
		log.Debugf("Location of Source Event %s at %s changed", a.Title, a.StartTime)
		return false
	}

	// Check if the reminders have changed
	// when the length does not match, we need to sync anyways
	if len(a.Reminders) != len(b.Reminders) {
		if len(a.Reminders) == 0 && len(b.Reminders) == 1 {
			log.Debugf("Count of Reminders in Source is 0 and in Sink is 1. Most of the time, this is due to the fact that the sink calendar has some reminder default configured. we're skipping here.")
		} else {
			log.Debugf("Count of Reminders of Source Event %s at %s changed", a.Title, a.StartTime)
			log.Debugf("Count of Reminders in Source: %d - in Sink: %d", len(a.Reminders), len(b.Reminders))
			return false
		}
	} else {
		sort.Sort(a.Reminders)
		sort.Sort(b.Reminders)

		// if the length does match, we need to check if the content was modified
		for i := 0; i < len(a.Reminders); i++ {
			if !a.Reminders[i].Equal(b.Reminders[i]) {
				log.Debugf("Reminder in Source is scheduled for %s, in the sink we have: %s", a.Reminders[i].Trigger.PointInTime.String(), b.Reminders[i].Trigger.PointInTime.String())
				return false
			}
		}
	}

	// Comparing the display name could be a problem because those are optional
	sort.Sort(a.Attendees)
	sort.Sort(b.Attendees)

	if len(a.Attendees) != len(b.Attendees) {
		log.Debugf("Amount of Attendees differ. Source: %d, Sink: %d", len(a.Attendees), len(b.Attendees))
		return false
	}

	for j := 0; j < len(a.Attendees); j++ {
		if !a.Attendees[j].Equal(b.Attendees[j]) {
			log.Debugf("Attendee in Source has Email: %s, in Sink the Email is: %s", a.Attendees[j].Email, b.Attendees[j].Email)
			log.Debugf("Attendee in Source has DisplayName: %s, in Sink the DisplayName is: %s", a.Attendees[j].DisplayName, b.Attendees[j].DisplayName)
			return false
		}
	}

	return true
}

type Calendar struct {
	ID          string
	Title       string
	Description string
}

type Attendees []Attendee

func (a Attendees) Len() int {
	return len(a)
}

func (a Attendees) Less(i, j int) bool {
	return a[i].Email < a[j].Email
}

func (a Attendees) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type Attendee struct {
	Email       string
	DisplayName string
}

func (a Attendee) Equal(value Attendee) bool {
	return a.Email == value.Email && a.DisplayName == value.DisplayName
}
