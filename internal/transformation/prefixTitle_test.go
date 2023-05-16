package transformation

import (
	"github.com/stretchr/testify/assert"
	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/models"
	"testing"
)

func TestPrefixTitle_Transform(t *testing.T) {
	tt := []struct {
		name                  string
		title                 string
		prefix                string
		expectedPrefixedTitle string
	}{
		{
			name:                  "prefix empty title with empty value",
			title:                 "",
			prefix:                "",
			expectedPrefixedTitle: "",
		},
		{
			name:                  "prefix empty title with value",
			title:                 "",
			prefix:                "bar",
			expectedPrefixedTitle: "bar",
		},
		{
			name:                  "prefix title with empty value",
			title:                 "foo",
			prefix:                "",
			expectedPrefixedTitle: "foo",
		},
		{
			name:                  "prefix title with value",
			title:                 "foo",
			prefix:                "bar_",
			expectedPrefixedTitle: "bar_foo",
		},
	}

	t.Parallel()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			expectedDescription := "description for testing"

			source := models.Event{
				Title:       tc.title,
				Description: "ignore this description",
			}
			sink := models.Event{
				Title:       tc.title,
				Description: expectedDescription,
			}

			transformer := PrefixTitle{Prefix: tc.prefix}

			event, err := transformer.Transform(source, sink)

			assert.Nil(t, err)

			expectedEvent := models.Event{
				Title:       tc.expectedPrefixedTitle,
				Description: expectedDescription,
			}
			assert.Equal(t, expectedEvent, event)

		})
	}

}
