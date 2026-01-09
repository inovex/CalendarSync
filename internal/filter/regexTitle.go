package filter

import (
	"regexp"

	"github.com/charmbracelet/log"

	"github.com/inovex/CalendarSync/internal/models"
)

type RegexTitle struct {
	ExcludeRegexp string
	IncludeRegexp string
}

func (a RegexTitle) Name() string {
	return "RegexTitle"
}

func (a RegexTitle) Filter(event models.Event) bool {
	// Filter() return true if the event should be kept

	// special case nothing excluded or included: keep all
	if len(a.ExcludeRegexp) == 0 && len(a.IncludeRegexp) == 0 {
		return true
	}

	// matchExclude defaults to false (exclude no events)
	matchExclude := false
	if len(a.ExcludeRegexp) > 0 {
		matchExclude = a.MatchRegex(event, a.ExcludeRegexp)
	}
	// matchInclude defaults to false (don't include)
	matchInclude := false
	if len(a.IncludeRegexp) > 0 {
		matchInclude = a.MatchRegex(event, a.IncludeRegexp)
	}

	// use allow - deny - allow pattern
	// (default is allow, then excluded are denied, but included are allowed again)
	if matchInclude && matchExclude {
		return true
	} else if matchExclude {
		return false
	} else {
		return true
	}
}

func (a RegexTitle) MatchRegex(event models.Event, re string) bool {
	log.Debugf("Running Regexp %s on event title: %s", a.ExcludeRegexp, event.Title)

	r, err := regexp.Compile(re)
	if err != nil {
		log.Fatalf("Regular expression of Filter %s is not valid, please check", a.Name())
	}

	// if the title matches the Regexp, return false (filter the event)
	if r.MatchString(event.Title) {
		log.Debugf("Regular Expression %s matches the events title: %s", a.Name(), event.Title)
		return true
	}

	return false
}
