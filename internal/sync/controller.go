package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/inovex/CalendarSync/internal/models"

	"github.com/charmbracelet/log"
)

// A Controller can synchronise the events from the sink via the given transformers into the sink
type Controller struct {
	source Source
	// transformers are applied in order
	transformers []Transformer
	sink         Sink
	concurrency  int
}

// NewController constructs a new Controller.
func NewController(source Source, sink Sink, transformer ...Transformer) Controller {
	return Controller{
		concurrency:  1,
		source:       source,
		transformers: transformer,
		sink:         sink,
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

	if log.GetLevel() == log.DebugLevel {
		for _, event := range source {
			log.Debug("source event loaded", "title", event.ShortTitle(), "time", event.StartTime)
		}
	}

	sink, err = p.sink.EventsInTimeframe(ctx, start, end)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get events in timeframe from sink %s: %v", p.sink.Name(), err)
	}

	if log.GetLevel() == log.DebugLevel {
		for _, event := range sink {
			log.Debug("sink event loaded", "title", event.ShortTitle(), "time", event.StartTime)
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

	// Transform source events before comparing them to the sink events
	transformedEventsInSource := []models.Event{}

	// Output which transformers were loaded
	for _, trans := range p.transformers {
		log.Info("loaded transformer", "name", trans.Name())
	}

	for _, event := range eventsInSource {
		transformedEventsInSource = append(transformedEventsInSource, TransformEvent(event, p.transformers...))
	}

	toCreate, toUpdate, toDelete := p.diffEvents(transformedEventsInSource, eventsInSink)
	if dryRun {
		log.Warn("we're running in dry run mode, no changes will be executed")
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
		if event.Metadata.SourceID == p.source.GetSourceID() {
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
			log.Info("new event, needs sync", "title", event.ShortTitle(), "time", event.StartTime.String())
			createEvents = append(createEvents, event)

		case sinkEvent.Metadata.SourceID != p.source.GetSourceID():
			log.Info("event was not synced by this source adapter, skipping", "title", event.ShortTitle(), "time", event.StartTime.String())

			// Only update the event if the event differs AND we synced it prior and set the correct metadata
		case !models.IsSameEvent(event, sinkEvent) && sinkEvent.Metadata.SourceID == p.source.GetSourceID():
			log.Info("event content changed, needs sync", "title", event.ShortTitle(), "time", event.StartTime.String())
			updateEvents = append(updateEvents, sinkEvent.Overwrite(event))

		default:
			log.Info("event in sync", "title", event.ShortTitle(), "time", event.StartTime.String())
		}
	}

	for _, event := range sinkEvents {
		if event.Metadata == nil {
			continue
		}
		_, exists := source[event.Metadata.SyncID]

		switch {
		case exists:
		case event.Metadata.SourceID == "":
			// An event which has not been synced correctly or has been synced prior to the SourceID implementation
			// should rather be removed. If the event still exists in the sourceEvents, it will eventually be re-synced.
			log.Info("event metadata corrupted (SourceID empty), deleting", "title", event.ShortTitle(), "time", event.StartTime.String())
			deleteEvents = append(deleteEvents, event)

		case event.Metadata.SourceID == p.source.GetSourceID():
			log.Info("sinkEvent is not (anymore) in sourceEvents, marked for removal", "title", event.ShortTitle(), "time", event.StartTime.String())
			deleteEvents = append(deleteEvents, event)

		default:
			// Do not delete events which were not loaded by the current sourceEvents-adapter.
			// This enables the synchronization of multiple sources without them interfering.
			log.Info("event is not in sourceEvents but was not synced with this source adapter, skipping", "title", event.ShortTitle(), "time", event.StartTime.String())
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
