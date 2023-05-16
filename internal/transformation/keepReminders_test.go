package transformation

import (
	"github.com/stretchr/testify/assert"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"
	"testing"
	"time"
)

func TestKeepReminders_Transform(t *testing.T) {
	var pointInTime = time.Now()

	tt := []struct {
		name              string
		sourceReminders   []models.Reminder
		sinkReminders     []models.Reminder
		expectedReminders []models.Reminder
	}{
		{
			name:              "keep empty reminders with empty source and empty sink",
			sourceReminders:   nil,
			sinkReminders:     nil,
			expectedReminders: nil,
		},
		{
			name:            "keep nil reminders with empty source and sink",
			sourceReminders: nil,
			sinkReminders: []models.Reminder{{
				Actions: models.ReminderActionDisplay,
				Trigger: models.ReminderTrigger{PointInTime: pointInTime},
			}},
			expectedReminders: nil,
		},
		{
			name:            "keep empty reminders with empty source and sink",
			sourceReminders: []models.Reminder{},
			sinkReminders: []models.Reminder{{
				Actions: models.ReminderActionDisplay,
				Trigger: models.ReminderTrigger{PointInTime: pointInTime},
			}},
			expectedReminders: []models.Reminder{},
		},
		{
			name:              "keep empty reminders with empty source and nil sink",
			sourceReminders:   []models.Reminder{},
			sinkReminders:     nil,
			expectedReminders: []models.Reminder{},
		},
		{
			name: "keep reminders with source and empty sink",
			sourceReminders: []models.Reminder{{
				Actions: models.ReminderActionDisplay,
				Trigger: models.ReminderTrigger{PointInTime: pointInTime},
			}},
			sinkReminders: nil,
			expectedReminders: []models.Reminder{{
				Actions: models.ReminderActionDisplay,
				Trigger: models.ReminderTrigger{PointInTime: pointInTime},
			}},
		},
		{
			name: "keep reminders with source and sink",
			sourceReminders: []models.Reminder{{
				Actions: models.ReminderActionDisplay,
				Trigger: models.ReminderTrigger{PointInTime: pointInTime},
			}},
			sinkReminders: []models.Reminder{{
				Actions: models.ReminderActionDisplay,
				Trigger: models.ReminderTrigger{PointInTime: pointInTime.Add(-time.Hour)},
			}},
			expectedReminders: []models.Reminder{{
				Actions: models.ReminderActionDisplay,
				Trigger: models.ReminderTrigger{PointInTime: pointInTime},
			}},
		},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expectedTitle := "title for testing"

			source := models.Event{
				Title:     "ignore this title",
				Reminders: tc.sourceReminders,
			}
			sink := models.Event{
				Title:     expectedTitle,
				Reminders: tc.sinkReminders,
			}

			var transformer KeepReminders
			event, err := transformer.Transform(source, sink)

			assert.NoError(t, err)
			expectedEvent := models.Event{
				Title:     expectedTitle,
				Reminders: tc.expectedReminders,
			}
			assert.Equal(t, expectedEvent, event)
		})
	}
}
