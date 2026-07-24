package adapter

import (
	"context"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/stretchr/testify/require"

	"github.com/inovex/CalendarSync/internal/config"
)

func TestNewSourceAdapterConfiguresBeforeOAuth(t *testing.T) {
	adapterConfig := config.NewAdapterConfig(config.Adapter{
		Type:     string(OutlookHttpCalendarType),
		Calendar: "calendar-id",
		Config:   config.CustomMap{"user": 42},
	})

	_, err := NewSourceAdapterFromConfig(context.Background(), 0, false, adapterConfig, nil, log.Default())

	require.EqualError(t, err, "Outlook adapter config 'user' must be a string")
}

func TestNewSinkAdapterConfiguresBeforeOAuth(t *testing.T) {
	adapterConfig := config.NewAdapterConfig(config.Adapter{
		Type:     string(OutlookHttpCalendarType),
		Calendar: "calendar-id",
		Config:   config.CustomMap{"user": 42},
	})

	_, err := NewSinkAdapterFromConfig(context.Background(), 0, false, adapterConfig, nil, log.Default())

	require.EqualError(t, err, "Outlook adapter config 'user' must be a string")
}
