package sync

import (
	"context"
	"testing"
	"time"

	"github.com/inovex/CalendarSync/internal/config"
	"github.com/inovex/CalendarSync/internal/models"

	"github.com/charmbracelet/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/inovex/CalendarSync/internal/sync/mocks"
)

type ControllerTestSuite struct {
	suite.Suite
	sink       *mocks.Sink
	source     *mocks.Source
	controller Controller
}

func (suite *ControllerTestSuite) SetupTest() {
	suite.sink = &mocks.Sink{}
	suite.source = &mocks.Source{}
	// Preconfigured Transformers for the tests
	transformers := TransformerFactory([]config.Transformer{
		{Name: "KeepLocation"},
		{Name: "KeepDescription"},
		{Name: "KeepTitle"},
		{Name: "KeepReminders"},
	})
	filters := FilterFactory([]config.Filter{
		{Name: "DeclinedEvents"},
	})
	suite.controller = NewController(log.Default(), suite.source, suite.sink, transformers, filters)
}

// TestDryRun tests that no acutal adapter func is called
func (suite *ControllerTestSuite) TestDryRun() {
	ctx := context.Background()
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour)

	sourceEvents := []models.Event{
		{
			ICalUID:     "testID",
			ID:          "testUID",
			Title:       "Title 1",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed1", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: startTime.Add(-10 * time.Minute)}}},
			Accepted:    true,
		},
		{
			ICalUID:     "testID3",
			ID:          "testUID3",
			Title:       "i will not be deleted in the sink",
			Description: "as this is a dry run",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed3", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: startTime.Add(-10 * time.Minute)}}},
			Accepted:    true,
		},
	}

	sinkEvents := []models.Event{
		{
			ICalUID:     "testID",
			ID:          "testUID",
			Title:       "Title 1",
			Description: "Description to be updated",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed1", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: startTime.Add(-10 * time.Minute)}}},
		},
		{
			ICalUID:     "testID2",
			ID:          "testUID2",
			Title:       "I will not get created",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed2", "uri", "sourceID"),
		},
	}
	suite.source.On("EventsInTimeframe", ctx, startTime, endTime).Return(sourceEvents, nil)
	suite.sink.On("EventsInTimeframe", ctx, startTime, endTime).Return(sinkEvents, nil)
	suite.sink.On("DeleteEvent", ctx, mock.AnythingOfType("models.Event")).Return(nil)
	suite.sink.On("GetCalendarID").Return("sinkID")
	suite.source.On("GetCalendarID").Return("sourceID")

	err := suite.controller.SynchroniseTimeframe(ctx, startTime, endTime, true)
	assert.NoError(suite.T(), err)

	suite.sink.AssertNotCalled(suite.T(), "CreateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNotCalled(suite.T(), "UpdateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNotCalled(suite.T(), "DeleteEvent", ctx, mock.AnythingOfType("models.Event"))
}

// TestCleanUp deletes all events synced by us in the sink calendar
func (suite *ControllerTestSuite) TestCleanUp() {
	ctx := context.Background()
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour)

	// can be empty as we throw them away in the cleanUp func anyways
	sourceEvents := []models.Event{}

	sinkEvents := []models.Event{
		{
			ICalUID:     "testID",
			ID:          "testUID",
			Title:       "Title 1",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed1", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: startTime.Add(-10 * time.Minute)}}},
		},
		{
			ICalUID:     "testID2",
			ID:          "testUID2",
			Title:       "Title 2",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed2", "uri", "sourceID"),
		},
		{
			ICalUID:     "testID3",
			ID:          "testUID3",
			Title:       "Title 3",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed3", "uri", "sourceID"),
		},
		{
			ICalUID:     "testID4",
			ID:          "testUID4",
			Title:       "Do not delete me",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
		},
		{
			ICalUID:     "testID5",
			ID:          "testUID5",
			Title:       "Do not delete me 2",
			Description: "i was synced from another source calendar",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed5", "uri", "sourceIDOfAnotherSourceCalendar"),
		},
	}
	expectedDelete := len(sinkEvents) - 2

	suite.source.On("EventsInTimeframe", ctx, startTime, endTime).Return(sourceEvents, nil)
	suite.sink.On("EventsInTimeframe", ctx, startTime, endTime).Return(sinkEvents, nil)
	suite.sink.On("DeleteEvent", ctx, mock.AnythingOfType("models.Event")).Return(nil)
	suite.sink.On("GetCalendarID").Return("sinkID")
	suite.source.On("GetCalendarID").Return("sourceID")

	err := suite.controller.CleanUp(ctx, startTime, endTime)
	assert.NoError(suite.T(), err)

	suite.sink.AssertNotCalled(suite.T(), "CreateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNotCalled(suite.T(), "UpdateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNumberOfCalls(suite.T(), "DeleteEvent", expectedDelete)
}

// TestCreateEventsEmptySink asserts that, given that the sink does not contain any events,
// the controller properly creates the events in the sink.
func (suite *ControllerTestSuite) TestCreateEventsEmptySink() {
	ctx := context.Background()
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour)
	eventsToCreate := []models.Event{
		{
			ICalUID:     "testID",
			ID:          "testUID",
			Title:       "Title",
			Description: "Description",
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(time.Hour),
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed1", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: time.Now().Add(-10 * time.Minute)}}},
			Accepted:    true,
		},
		{
			ICalUID:     "testID2",
			ID:          "testUID2",
			Title:       "Title",
			Description: "Description",
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(time.Hour),
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed2", "uri", "sourceID"),
			Accepted:    true,
		},
	}

	suite.source.On("EventsInTimeframe", ctx, startTime, endTime).Return(eventsToCreate, nil)
	suite.sink.On("EventsInTimeframe", ctx, startTime, endTime).Return(nil, nil)
	suite.sink.On("CreateEvent", ctx, mock.AnythingOfType("models.Event")).Return(nil)
	suite.sink.On("GetCalendarID").Return("sinkID")

	err := suite.controller.SynchroniseTimeframe(ctx, startTime, endTime, false)
	assert.NoError(suite.T(), err)

	suite.source.AssertCalled(suite.T(), "EventsInTimeframe", ctx, startTime, endTime)
	suite.sink.AssertCalled(suite.T(), "EventsInTimeframe", ctx, startTime, endTime)
	suite.sink.AssertNumberOfCalls(suite.T(), "CreateEvent", len(eventsToCreate))
	suite.sink.AssertNotCalled(suite.T(), "UpdateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNotCalled(suite.T(), "DeleteEvent", ctx, mock.AnythingOfType("models.Event"))
}

// TestDeleteEventsNotInSink verifies that if events are present in the sink-adapter, but not in the source, these
// events are deleted in the sink.
func (suite *ControllerTestSuite) TestDeleteEventsNotInSink() {
	ctx := context.Background()
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour)
	sourceEvents := []models.Event{
		{
			ICalUID:     "testID",
			ID:          "testUID",
			Title:       "Title 1",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed1", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: startTime.Add(-10 * time.Minute)}}},
			Accepted:    true,
		},
	}
	sinkEvents := []models.Event{
		// Will not get deleted
		{
			ICalUID:     "testID",
			ID:          "testUID",
			Title:       "Title 1",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed1", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: startTime.Add(-10 * time.Minute)}}},
		},
		// Will get deleted
		{
			ICalUID:     "testID2",
			ID:          "testUID2",
			Title:       "Title 2",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed2", "uri", "sourceID"),
		},
		// Will get deleted
		{
			ICalUID:     "testID3",
			ID:          "testUID3",
			Title:       "Title 3",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed3", "uri", "sourceID"),
		},
		// Will not get deleted, as it does not contain metadata
		{
			ICalUID:     "testID4",
			ID:          "testUID4",
			Title:       "i should not get deleted",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
		},
	}
	expectedDelete := 2

	suite.source.On("EventsInTimeframe", ctx, startTime, endTime).Return(sourceEvents, nil)
	suite.sink.On("EventsInTimeframe", ctx, startTime, endTime).Return(sinkEvents, nil)
	suite.sink.On("DeleteEvent", ctx, mock.AnythingOfType("models.Event")).Return(nil)
	// UpdateEvent gets called because the remaining event in the sink will get updated because there are no transformers configured
	suite.sink.On("UpdateEvent", ctx, mock.AnythingOfType("models.Event")).Return(nil)
	suite.sink.On("GetCalendarID").Return("sinkID")
	suite.source.On("GetCalendarID").Return("sourceID")

	err := suite.controller.SynchroniseTimeframe(ctx, startTime, endTime, false)
	assert.NoError(suite.T(), err)

	suite.sink.AssertNotCalled(suite.T(), "CreateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNotCalled(suite.T(), "UpdateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNumberOfCalls(suite.T(), "DeleteEvent", expectedDelete)
}

// TestDoNotResurrectEvents verifies that if events are present in the source-adapter that originated
// from the sink, but have been deleted there, these events are not copied to the sink.
// This ensures that for two calendars A and B, with sync A->B and B->A, that an event
// will not be restored if the original event was deleted.
func (suite *ControllerTestSuite) TestDoNotResurrectEvents() {
	ctx := context.Background()
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour)
	sourceEvents := []models.Event{
		{
			ICalUID:     "testID",
			ID:          "testUID",
			Title:       "Title 1",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			// originates from sink
			Metadata:  models.NewEventMetadata("seed1", "uri", "sinkID"),
			Reminders: []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: startTime.Add(-10 * time.Minute)}}},
		},
	}
	sinkEvents := []models.Event{}

	suite.source.On("EventsInTimeframe", ctx, startTime, endTime).Return(sourceEvents, nil)
	suite.sink.On("EventsInTimeframe", ctx, startTime, endTime).Return(sinkEvents, nil)
	suite.sink.On("GetCalendarID").Return("sinkID")
	suite.source.On("GetCalendarID").Return("sourceID")

	err := suite.controller.SynchroniseTimeframe(ctx, startTime, endTime, false)
	assert.NoError(suite.T(), err)

	suite.sink.AssertNotCalled(suite.T(), "CreateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNotCalled(suite.T(), "UpdateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNotCalled(suite.T(), "DeleteEvent", ctx, mock.AnythingOfType("models.Event"))
}

// TestUpdateEventsPrefilledSink asserts that, given that the sink does contain any events,
// the controller properly updates the events in the sink.
// and leaves unmanaged events as they are
func (suite *ControllerTestSuite) TestUpdateEventsPrefilledSink() {
	ctx := context.Background()
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour)

	sourceEvents := []models.Event{
		{
			ICalUID:     "testID",
			ID:          "testUID",
			Title:       "Update-Test-Title-1",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed1", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: time.Now().Add(-10 * time.Minute)}}},
			Accepted:    true,
		},
		{
			ICalUID:     "testID2",
			ID:          "testUID2",
			Title:       "Update-Test-Title-2",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed2", "uri", "sourceID"),
			Accepted:    true,
		},
		{
			ICalUID:     "testID3",
			ID:          "testUID3",
			Title:       "Update-Test-Title-3-Modified",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed3", "uri", "sourceID"),
			Accepted:    true,
		},
		{
			ICalUID:     "testID4",
			ID:          "testUID4",
			Title:       "Update-Test-Title-4",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed4", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: time.Now().Add(-30 * time.Minute)}}},
			Accepted:    true,
		},
	}

	// the sink events are missing some fields of the source events (e.g. the reminders in the first event)
	sinkEvents := []models.Event{
		// This event should get updated, as the reminder is missing
		{
			ICalUID:     "testID",
			ID:          "testUID",
			Title:       "Update-Test-Title-1",
			Description: "Description",
			StartTime:   sourceEvents[0].StartTime,
			EndTime:     sourceEvents[0].EndTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed1", "uri", "sourceID"),
		},
		// This event should NOT get updated
		{
			ICalUID:     "testID2",
			ID:          "testUID2",
			Title:       "Update-Test-Title-2",
			Description: "Description",
			StartTime:   sourceEvents[1].StartTime,
			EndTime:     sourceEvents[1].EndTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed2", "uri", "sourceID"),
		},
		// This event should be updated as the title was modified in the source
		{
			ICalUID:     "testID3",
			ID:          "testUID3",
			Title:       "Update-Test-Title-3",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed3", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: time.Now().Add(-10 * time.Minute)}}},
		},
		// This event should be updated as the reminder time in the source in 30 mins and here we got 10 mins
		{
			ICalUID:     "testID4",
			ID:          "testUID4",
			Title:       "Update-Test-Title-4",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed4", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: time.Now().Add(-10 * time.Minute)}}},
		},
		// This event should NOT get updated, as this is an event we do not manage
		{
			ICalUID:     "testID5",
			ID:          "testUID5",
			Title:       "Do not update or delete me",
			Description: "Description",
			StartTime:   startTime,
			EndTime:     endTime,
			AllDay:      false,
		},
	}

	// you may want to change this, based on the test cases
	var eventsToBeUpdated = 3

	suite.source.On("EventsInTimeframe", ctx, startTime, endTime).Return(sourceEvents, nil)
	suite.sink.On("EventsInTimeframe", ctx, startTime, endTime).Return(sinkEvents, nil)
	suite.sink.On("UpdateEvent", ctx, mock.AnythingOfType("models.Event")).Return(nil)
	suite.sink.On("GetCalendarID").Return("sinkID")
	suite.source.On("GetCalendarID").Return("sourceID")

	err := suite.controller.SynchroniseTimeframe(ctx, startTime, endTime, false)
	assert.NoError(suite.T(), err)

	suite.source.AssertCalled(suite.T(), "EventsInTimeframe", ctx, startTime, endTime)
	suite.sink.AssertCalled(suite.T(), "EventsInTimeframe", ctx, startTime, endTime)
	suite.sink.AssertNumberOfCalls(suite.T(), "UpdateEvent", eventsToBeUpdated)
	suite.sink.AssertNotCalled(suite.T(), "CreateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNotCalled(suite.T(), "DeleteEvent", ctx, mock.AnythingOfType("models.Event"))
}

// TestCreateEventsDeclined asserts that, only the accepted event gets synced
func (suite *ControllerTestSuite) TestCreateEventsDeclined() {
	ctx := context.Background()
	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour)
	eventsToCreate := []models.Event{
		{
			ICalUID:     "testID",
			ID:          "testUID",
			Title:       "Title",
			Description: "Description",
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(time.Hour),
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed1", "uri", "sourceID"),
			Reminders:   []models.Reminder{{Actions: models.ReminderActionDisplay, Trigger: models.ReminderTrigger{PointInTime: time.Now().Add(-10 * time.Minute)}}},
			Accepted:    true,
		},
		{
			ICalUID:     "testID2",
			ID:          "testUID2",
			Title:       "Title",
			Description: "Description",
			StartTime:   time.Now(),
			EndTime:     time.Now().Add(time.Hour),
			AllDay:      false,
			Metadata:    models.NewEventMetadata("seed2", "uri", "sourceID"),
			Accepted:    false,
		},
	}

	suite.source.On("EventsInTimeframe", ctx, startTime, endTime).Return(eventsToCreate, nil)
	suite.sink.On("EventsInTimeframe", ctx, startTime, endTime).Return(nil, nil)
	suite.sink.On("CreateEvent", ctx, mock.AnythingOfType("models.Event")).Return(nil)
	suite.sink.On("GetCalendarID").Return("sinkID")

	err := suite.controller.SynchroniseTimeframe(ctx, startTime, endTime, false)
	assert.NoError(suite.T(), err)

	suite.source.AssertCalled(suite.T(), "EventsInTimeframe", ctx, startTime, endTime)
	suite.sink.AssertCalled(suite.T(), "EventsInTimeframe", ctx, startTime, endTime)
	suite.sink.AssertNumberOfCalls(suite.T(), "CreateEvent", len(eventsToCreate)-1)
	suite.sink.AssertNotCalled(suite.T(), "UpdateEvent", ctx, mock.AnythingOfType("models.Event"))
	suite.sink.AssertNotCalled(suite.T(), "DeleteEvent", ctx, mock.AnythingOfType("models.Event"))
}

func TestControllerTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}

func TestController_diffEvents(t *testing.T) {
	tt := []struct {
		name                 string
		sink                 []models.Event
		source               []models.Event
		expectedCreateEvents []models.Event
		expectedUpdateEvents []models.Event
		expectedDeleteEvents []models.Event
	}{
		{
			name:                 "should return empty lists when non-existing sink and non-existing source",
			sink:                 nil,
			source:               nil,
			expectedCreateEvents: []models.Event{},
			expectedUpdateEvents: []models.Event{},
			expectedDeleteEvents: []models.Event{},
		},
		{
			name:                 "should return empty lists when empty sink and empty source",
			sink:                 []models.Event{},
			source:               []models.Event{},
			expectedCreateEvents: []models.Event{},
			expectedUpdateEvents: []models.Event{},
			expectedDeleteEvents: []models.Event{},
		},
		{
			name: "should return create list when only create event exists in source",
			source: []models.Event{{
				Metadata: &models.Metadata{
					SyncID:   "unicorn",
					SourceID: "sourceID",
				},
				Title: "Foo",
			}},
			expectedCreateEvents: []models.Event{{
				Metadata: &models.Metadata{
					SyncID:   "unicorn",
					SourceID: "sourceID",
				},
				Title: "Foo",
			}},
			expectedUpdateEvents: []models.Event{},
			expectedDeleteEvents: []models.Event{},
		},
		{
			name: "should return update list when updated event exists in source and out-dated event in sink, leaves the unmanaged sink event alone",
			source: []models.Event{{
				Metadata: &models.Metadata{
					SyncID:   "unicorn",
					SourceID: "sourceID",
				},
				Title: "Bar",
			}},
			sink: []models.Event{{
				Metadata: &models.Metadata{
					SyncID:   "unicorn",
					SourceID: "sourceID",
				},
				Title: "Foo",
			},
				{
					Title: "Bar",
				}},
			expectedUpdateEvents: []models.Event{{
				Metadata: &models.Metadata{
					SyncID:   "unicorn",
					SourceID: "sourceID",
				},
				Title: "Bar",
			}},
			expectedCreateEvents: []models.Event{},
			expectedDeleteEvents: []models.Event{},
		},
		{
			name: "should return delete list when event not exists in source but is included in sink",
			sink: []models.Event{{
				Metadata: &models.Metadata{
					SyncID:   "unicorn",
					SourceID: "sourceID",
				},
				Title: "Foo",
			}},
			expectedDeleteEvents: []models.Event{{
				Metadata: &models.Metadata{
					SyncID:   "unicorn",
					SourceID: "sourceID",
				},
				Title: "Foo",
			}},
			expectedCreateEvents: []models.Event{},
			expectedUpdateEvents: []models.Event{},
		},
		{
			name: "should return delete list when events not exists in source but in sink",
			sink: []models.Event{
				{
					Title: "Delete me",
					Metadata: &models.Metadata{
						SourceID: "sourceID",
					},
				},
			},
			expectedDeleteEvents: []models.Event{
				{
					Title: "Delete me",
					Metadata: &models.Metadata{
						SourceID: "sourceID",
					},
				},
			},
			expectedCreateEvents: []models.Event{},
			expectedUpdateEvents: []models.Event{},
		},
		{
			name: "should return delete list when sourceId is set in sink",
			sink: []models.Event{
				{
					Title: "Don't delete me",
					Metadata: &models.Metadata{
						SourceID: "Foo",
					},
				},
			},
			expectedDeleteEvents: []models.Event{},
			expectedCreateEvents: []models.Event{},
			expectedUpdateEvents: []models.Event{},
		},
		{
			name: "should return create, update and delete list when different events exists in source and in sink",
			source: []models.Event{
				{
					Metadata: &models.Metadata{
						SyncID:   "create",
						SourceID: "sourceID",
					},
					Title: "create",
				},
				{
					Metadata: &models.Metadata{
						SyncID:   "update",
						SourceID: "sourceID",
					},
					Title:       "update",
					Description: "v2.0.1",
				},
			},
			sink: []models.Event{
				{
					Metadata: &models.Metadata{
						SyncID:   "update",
						SourceID: "sourceID",
					},
					Title: "update",
				},
				{
					Metadata: &models.Metadata{
						SyncID:   "delete",
						SourceID: "sourceID",
					},
					Title: "delete",
				},
			},
			expectedCreateEvents: []models.Event{
				{
					Metadata: &models.Metadata{
						SyncID:   "create",
						SourceID: "sourceID",
					},
					Title: "create",
				},
			},
			expectedUpdateEvents: []models.Event{
				{
					Metadata: &models.Metadata{
						SyncID:   "update",
						SourceID: "sourceID",
					},
					Title:       "update",
					Description: "v2.0.1",
				},
			},
			expectedDeleteEvents: []models.Event{
				{
					Metadata: &models.Metadata{
						SyncID:   "delete",
						SourceID: "sourceID",
					},
					Title: "delete",
				},
			},
		},
		{
			name:   "should leave unmanaged events alone",
			source: []models.Event{},
			sink: []models.Event{
				{
					Title: "leave me alone",
				},
				{
					Metadata: &models.Metadata{
						SyncID:   "delete",
						SourceID: "sourceID1337",
					},
					Title: "leave me alone as my source id is different",
				},
			},
			expectedCreateEvents: []models.Event{},
			expectedUpdateEvents: []models.Event{},
			expectedDeleteEvents: []models.Event{},
		},
		{
			name: "should not resurrect events already deleted from source when syncing back from sink",
			source: []models.Event{{
				Metadata: &models.Metadata{
					SyncID:   "unicorn",
					SourceID: "sinkID",
				},
				Title: "Foo",
			}},
			expectedCreateEvents: []models.Event{},
			expectedUpdateEvents: []models.Event{},
			expectedDeleteEvents: []models.Event{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var source mocks.Source
			source.On("GetCalendarID").Return("sourceID")

			var sink mocks.Sink
			sink.On("GetCalendarID").Return("sinkID")

			var controller = Controller{
				source: &source,
				sink:   &sink,
				logger: log.Default(),
			}

			creates, updates, deletes := controller.diffEvents(tc.source, tc.sink)

			assert.Equal(t, tc.expectedCreateEvents, creates)
			assert.Equal(t, tc.expectedUpdateEvents, updates)
			assert.Equal(t, tc.expectedDeleteEvents, deletes)
		})
	}
}
