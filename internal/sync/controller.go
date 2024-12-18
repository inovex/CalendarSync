package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/inovex/CalendarSync/internal/models"

	"github.com/charmbracelet/log"
)

var (
	logFields = func(event models.Event) []any {
		return []any{"title", event.ShortTitle(), "time", event.StartTime.String()}
	}
)

// A Controller can synchronise the events from the sink via the given transformers into the sink
type Controller struct {
	source Source
	// transformers are applied in order
	transformers []Transformer
	filters      []Filter
	sink         Sink
	concurrency  int
	logger       *log.Logger
}

// NewController constructs a new Controller.
func NewController(logger *log.Logger, source Source, sink Sink, transformer []Transformer, filters []Filter) Controller {
	return Controller{
		concurrency:  1,
		source:       source,
		transformers: transformer,
		filters:      filters,
		sink:         sink,
		logger:       logger,
	}
}

func (p *Controller) SetConcurrency(concurrency int) {
	p.concurrency = concurrency
}

// loadEvents will load source and sink events in the given timeframe and return them
func (p Controller) loadEvents(ctx context.Context, start, end time.Time) (source []models.Event, sink []models.Event, err error) {
	source, err = p.source.EventsInTimeframe(ctx, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get events in timeframe from source %s: %v", p.source.Name(), err)
	}

	if p.logger.GetLevel() == log.DebugLevel {
		for _, event := range source {
			p.logger.Debug("source event loaded", logFields(event)...)
		}
	}

	sink, err = p.sink.EventsInTimeframe(ctx, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get events in timeframe from sink %s: %v", p.sink.Name(), err)
	}

	if p.logger.GetLevel() == log.DebugLevel {
		for _, event := range sink {
			p.logger.Debug("sink event loaded", logFields(event)...)
		}
	}

	return source, sink, err
}

// SynchroniseTimeframe synchronises all events in the given timeframe
func (p Controller) SynchroniseTimeframe(ctx context.Context, start time.Time, end time.Time, dryRun bool) error {
	eventsInSource, eventsInSink, err := p.loadEvents(ctx, start, end)
	if err != nil {
		return err
	}

	filteredEventsInSource := []models.Event{}

	for _, filter := range p.filters {
		p.logger.Debug("loaded filter", "name", filter.Name())
	}

	for _, event := range eventsInSource {
		if FilterEvent(event, p.filters...) {
			filteredEventsInSource = append(filteredEventsInSource, event)
		} else {
			p.logger.Debug("filter rejects event", logFields(event)...)
		}
	}

	// Transform source events before comparing them to the sink events
	transformedEventsInSource := []models.Event{}

	// Output which transformers were loaded
	for _, trans := range p.transformers {
		p.logger.Debug("loaded transformer", "name", trans.Name())
	}

	for _, event := range filteredEventsInSource {
		transformedEventsInSource = append(transformedEventsInSource, TransformEvent(event, p.transformers...))
	}

	toCreate, toUpdate, toDelete := p.diffEvents(transformedEventsInSource, eventsInSink)
	log.Infof("found %d new, %d changed, and %d deleted events", len(toCreate), len(toUpdate), len(toDelete))
	if dryRun {
		p.logger.Warn("we're running in dry run mode, no changes will be executed")
		return nil
	}

	var tasks []taskFunc
	for _, event := range toDelete {
		// redefine to let the closure capture individual variables
		event := event
		tasks = append(tasks, func() error {
			if err := p.sink.DeleteEvent(ctx, event); err != nil {
				return fmt.Errorf("failed to delete event %s in sink %s: %w", event.ShortTitle(), p.sink.Name(), err)
			}
			return nil
		})
	}

	for _, event := range toCreate {
		// redefine to let the closure capture individual variables
		event := event
		tasks = append(tasks, func() error {
			if err := p.sink.CreateEvent(ctx, event); err != nil {
				return fmt.Errorf("failed to create event %s in sink %s: %w", event.ShortTitle(), p.sink.Name(), err)
			}
			return nil
		})
	}

	for _, event := range toUpdate {
		// redefine to let the closure capture individual variables
		event := event
		tasks = append(tasks, func() error {
			if err := p.sink.UpdateEvent(ctx, event); err != nil {
				return fmt.Errorf("unable to update event %s: %s at %s in sink %s: %v", event.ShortTitle(), event.ShortTitle(), event.StartTime.Format(time.RFC1123), p.sink.Name(), err)
			}
			return nil
		})
	}

	return parallel(ctx, p.concurrency, tasks)
}

func (p Controller) CleanUp(ctx context.Context, start time.Time, end time.Time) error {
	_, eventsInSink, err := p.loadEvents(ctx, start, end)
	if err != nil {
		return err
	}

	sink := maps(eventsInSink)

	var tasks []taskFunc

	for _, event := range sink {
		// Check if the sink event was synced by us, if there's no metadata the event may
		// be there because we were invited or because it is not managed by us
		if event.Metadata.SourceID == p.source.GetCalendarHash() {
			// redefine to let the closure capture individual variables
			event := event
			tasks = append(tasks, func() error {
				if err := p.sink.DeleteEvent(ctx, event); err != nil {
					return fmt.Errorf("failed to delete event %s in sink %s: %w", event.ShortTitle(), p.sink.Name(), err)
				}
				return nil
			})
		}
	}

	return parallel(ctx, p.concurrency, tasks)
}

func (p Controller) diffEvents(sourceEvents []models.Event, sinkEvents []models.Event) ([]models.Event, []models.Event, []models.Event) {
	var (
		createEvents = make([]models.Event, 0)
		updateEvents = make([]models.Event, 0)
		deleteEvents = make([]models.Event, 0)
	)

	source := maps(sourceEvents)
	sink := maps(sinkEvents)

	for _, event := range sourceEvents {
		if event.Metadata == nil {
			continue
		}

		sinkEvent, exists := sink[event.Metadata.SyncID]

		switch {
		case !exists:
			// Don't sync synced events back to their original calendar to prevent resurrecting
			// deleted events.
			// Problem:
			// - Sync event (from calendar A) to create eventCopy (calendar B, SourceID = calendar A).
			// - Delete event (in calendar A)
			// - Run sync from calendar B to calendar A. This will copy (and thereby resurrect) the event.
			//
			// Solution: Ignore events that originate from the sink, but no longer exist there.
			if event.Metadata.SourceID == p.sink.GetCalendarHash() {
				p.logger.Info("skipping event as it originates from the sink, but no longer exists there", logFields(event)...)
				continue
			}
			p.logger.Info("new event, needs sync", logFields(event)...)
			createEvents = append(createEvents, event)

		case sinkEvent.Metadata.SourceID != p.source.GetCalendarHash():
			p.logger.Info("event was not synced by this source adapter, skipping", logFields(event)...)

			// Only update the event if the event differs AND we synced it prior and set the correct metadata
		case !models.IsSameEvent(event, sinkEvent) && sinkEvent.Metadata.SourceID == p.source.GetCalendarHash():
			p.logger.Info("event content changed, needs sync", logFields(event)...)
			updateEvents = append(updateEvents, sinkEvent.Overwrite(event))

		default:
			p.logger.Debug("event in sync", logFields(event)...)
		}
	}

	for _, event := range sinkEvents {
		if event.Metadata == nil {
			continue
		}
		_, exists := source[event.Metadata.SyncID]

		switch {
		case exists:
			// Nothing to do

		case event.Metadata.SourceID == p.source.GetCalendarHash():
			p.logger.Info("sinkEvent is not (anymore) in sourceEvents, marked for removal", logFields(event)...)
			deleteEvents = append(deleteEvents, event)

		default:
			// Do not delete events which were not loaded by the current sourceEvents-adapter.
			// This enables the synchronization of multiple sources without them interfering.
			p.logger.Debug("event is not in sourceEvents but was not synced with this source adapter, skipping", logFields(event)...)
		}
	}

	return createEvents, updateEvents, deleteEvents
}

func maps(events []models.Event) map[string]models.Event {
	result := make(map[string]models.Event)
	for _, event := range events {
		if event.Metadata != nil {
			result[event.Metadata.SyncID] = event
		}
	}
	return result
}
