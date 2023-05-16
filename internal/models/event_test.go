package models

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEvent_Sync(t *testing.T) {
	startTime := time.Now()
	endTime := time.Now().Add(time.Hour)

	tests := []struct {
		name          string
		dest          Event
		source        Event
		expectedEvent Event
	}{
		{
			name: "overwrite dest event with source",
			dest: Event{
				ICalUID:     "foo",
				ID:          "bar",
				Title:       "Dest",
				Description: "Should stay",
				StartTime:   time.Now(),
				EndTime:     time.Now().Add(2 * time.Hour),
				AllDay:      false,
				Metadata: &Metadata{
					SyncID: "foo",
				},
			},
			source: Event{
				ICalUID:     "New ID",
				ID:          "New UUID",
				Title:       "Source",
				Description: "Should become",
				StartTime:   startTime,
				EndTime:     endTime,
				AllDay:      true,
				Metadata: &Metadata{
					SyncID: "foo",
				},
			},
			expectedEvent: Event{
				ICalUID:     "foo",
				ID:          "bar",
				Title:       "Source",
				Description: "Should become",
				StartTime:   startTime,
				EndTime:     endTime,
				AllDay:      true,
				Metadata: &Metadata{
					SyncID: "foo",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.dest.Overwrite(tt.source)

			assert.Equal(t, tt.expectedEvent, actual)
		})
	}
}

func TestReminders_Sort(t *testing.T) {
	now := time.Now()
	tt := []struct {
		name      string
		Reminders Reminders
	}{
		{
			name: "Sort all reminders",
			Reminders: Reminders{
				{
					Actions: 3,
					Trigger: ReminderTrigger{
						PointInTime: now.Add(time.Hour),
					},
				},
				{
					Actions: 2,
					Trigger: ReminderTrigger{
						PointInTime: now.Add(time.Minute),
					},
				},
				{
					Actions: 1,
					Trigger: ReminderTrigger{
						PointInTime: now.Add(time.Second),
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			sort.Sort(&tc.Reminders)

			for i := 1; i <= 3; i++ {
				assert.Equal(t, ReminderActions(i), tc.Reminders[i-1].Actions)
			}
		})
	}
}
