package outlook_http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// flattenLocation must leave a plain display-name location untouched (nil
// address) and must not duplicate a component already present in the name.
func TestFlattenLocation(t *testing.T) {
	assert.Equal(t, "Conference Room A",
		flattenLocation(Location{Name: "Conference Room A"}),
		"a nil address must return the display name unchanged")

	// Venue name with a disjoint street address: street must survive.
	got := flattenLocation(Location{Name: "Harry's Bar", Address: &PhysicalAddress{
		Street: "123 Main St", City: "Springfield", State: "IL", PostalCode: "62704",
	}})
	assert.Equal(t, "Harry's Bar, 123 Main St, Springfield, IL, 62704", got)

	// displayName already holds the street: it must not be repeated.
	got = flattenLocation(Location{Name: "123 Main St", Address: &PhysicalAddress{
		Street: "123 Main St", City: "Springfield", State: "IL", PostalCode: "62704",
	}})
	assert.Equal(t, "123 Main St, Springfield, IL, 62704", got)
}
