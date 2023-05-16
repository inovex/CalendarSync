package auth

import (
	"context"
	"fmt"

	"gitlab.inovex.de/inovex-calendarsync/calendarsync/internal/config"
)

type Storage interface {
	WriteCalendarAuth(CalendarAuth) (bool, error)
	ReadCalendarAuth(calendarID string) (*CalendarAuth, error)
	RemoveCalendarAuth(calendarID string) error
	Setup(config config.AuthStorage, encryptionPassphrase string) error
}

type CalendarAuth struct {
	CalendarID string
	OAuth2     OAuth2Object
}

type OAuth2Object struct {
	AccessToken  string
	RefreshToken string
	Expiry       string
	TokenType    string
}

func StorageFactory(typ string) (Storage, error) {
	switch typ {
	case "yaml":
		return new(YamlStorage), nil
	default:
		return nil, fmt.Errorf("unknown storage mode %s", typ)
	}
}

func NewStorageAdapterFromConfig(ctx context.Context, config config.AuthStorage, encryptionPassphrase string) (Storage, error) {
	storage, err := StorageFactory(config.StorageMode)
	if err != nil {
		return nil, err
	}

	err = storage.Setup(config, encryptionPassphrase)
	if err != nil {
		return nil, err
	}

	return storage, nil
}
