package filter

import (
	"regexp"

	"github.com/charmbracelet/log"

	"github.com/inovex/CalendarSync/internal/models"
)

type RegexTitle struct {
	Regexp string
}

func (a RegexTitle) Name() string {
	return "RegexTitle"
}

func (a RegexTitle) Filter(event models.Event) bool {

	if len(a.Regexp) == 0 {
		log.Debugf("Regular Expression is empty, skipping Filter %s for event: %s", a.Name(), event.Title)
		return true
	}

	log.Debugf("Running Regexp %s on event title: %s", a.Regexp, event.Title)

	r, err := regexp.Compile(a.Regexp)
	if err != nil {
		log.Fatalf("Regular expression of Filter %s is not valid, please check", a.Name())
	}

	// if the title matches the Regexp, return false (filter the event)
	if r.MatchString(event.Title) {
		log.Debugf("Regular Expression %s matches the events title: %s, gets filtered", a.Name(), event.Title)
		return false
	}

	return true
}
