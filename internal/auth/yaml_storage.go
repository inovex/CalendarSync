package auth

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/charmbracelet/log"
	"github.com/inovex/CalendarSync/internal/config"
	"gopkg.in/yaml.v3"
)

type YamlStorage struct {
	StoragePath          string
	StorageEncryptionKey string
	// Holds the decrypted CalendarAuth Config in memory, so the file does not have to be read multiple times
	CachedAuth []CalendarAuth
}

func (y *YamlStorage) Setup(config config.AuthStorage, encryptionPassphrase string) error {
	y.StorageEncryptionKey = encryptionPassphrase
	y.StoragePath = config.Config["path"].(string)
	return nil
}

func (y *YamlStorage) WriteCalendarAuth(newCal CalendarAuth) (bool, error) {
	file, err := y.readAndParseFile()
	err = ignoreNoFile(err)
	if err != nil {
		return false, err
	}

	cals := []CalendarAuth{}
	if file != nil {
		// Iterate through all stored calendar auth objects and only append those with a different ID to the new slice
		// Calendar ID is supposed to be unique and old entries will be "overwritten"
		for _, cal := range file.Calendars {
			if cal.CalendarID != newCal.CalendarID {
				cals = append(cals, cal)
			}
		}
	}

	cals = append(cals, newCal)
	err = y.writeFile(cals)
	if err != nil {
		return false, err
	}

	// Adding freshly written CalendarAuth to memory
	// Probably unneeded, the next time this data will be retrieved is on the next calendarsync run
	log.Debugf("Adding calendar auth for cal %s to memory", newCal.CalendarID)
	y.DecryptedAuth = append(y.DecryptedAuth, newCal)

	return true, nil
}

func (y *YamlStorage) ReadCalendarAuth(calendarID string) (*CalendarAuth, error) {
	// if we already decrypted the file, read from memory
	if len(y.DecryptedAuth) > 0 {
		for _, cal := range y.DecryptedAuth {
			if cal.CalendarID == calendarID {
				log.Debug("loaded auth data from memory", "calendarID", cal.CalendarID)
				return &cal, nil
			}
		}
	}

	file, err := y.readAndParseFile()
	if err != nil {
		return nil, ignoreNoFile(err)
	}

	for _, cal := range file.Calendars {
		if cal.CalendarID == calendarID {
			return &cal, nil
		}
	}

	return nil, nil
}

func (y *YamlStorage) RemoveCalendarAuth(calendarID string) error {
	file, err := y.readAndParseFile()
	err = ignoreNoFile(err)
	if err != nil {
		return err
	}

	cals := []CalendarAuth{}
	if file != nil {
		// Iterate through all stored calendar auth objects and only append those with a different ID to the new slice
		// The provided ID will be removed
		for _, cal := range file.Calendars {
			if cal.CalendarID != calendarID {
				cals = append(cals, cal)
			}
		}
	}

	err = y.writeFile(cals)
	if err != nil {
		return err
	}

	return nil
}

func (y *YamlStorage) writeFile(cals []CalendarAuth) error {
	var writer io.Writer
	file, err := os.OpenFile(y.StoragePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open storage file: %w", err)
	}
	defer file.Close()

	writer = file

	// if encryption is enabled encrypt data
	if y.StorageEncryptionKey != "" {
		eFile := NewEncryptedFile(file, y.StorageEncryptionKey)
		defer eFile.Close()
		writer = eFile
	}

	err = yaml.NewEncoder(writer).Encode(storageFile{Calendars: cals})
	if err != nil {
		return fmt.Errorf("cannot write calendar auth data: %w", err)
	}

	return nil
}

func (y *YamlStorage) readAndParseFile() (*storageFile, error) {
	file, err := os.Open(y.StoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage path: %w", err)
	}
	defer file.Close()

	var reader io.Reader
	reader = file

	storageIsEncrypted, err := isEncrypted(file)
	if err != nil {
		return nil, err
	}

	if y.StorageEncryptionKey == "" && storageIsEncrypted {
		return nil, fmt.Errorf("no encryption key provided, but auth storage is encrypted")
	}

	// if encryption is enabled decrypt data
	if y.StorageEncryptionKey != "" && storageIsEncrypted {
		reader = NewEncryptedFile(file, y.StorageEncryptionKey)
	}

	var data = &storageFile{}
	err = yaml.NewDecoder(reader).Decode(data)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal config file: %w", err)
	}

	// write encrypt storage if encryption key is provided first time
	if !storageIsEncrypted && y.StorageEncryptionKey != "" {
		err := y.writeFile(data.Calendars)
		if err != nil {
			return nil, err
		}
	}

	return data, nil
}

func isEncrypted(reader io.ReaderAt) (bool, error) {
	encryptedStart := []byte("age-encryption.org")
	actualStart := make([]byte, len(encryptedStart))
	_, err := reader.ReadAt(actualStart, 0)
	if err != nil {
		return false, fmt.Errorf("failed to read auth-storage reader: %w", err)
	}
	storageIsEncrypted := bytes.Equal(actualStart, encryptedStart)
	return storageIsEncrypted, nil
}

func ignoreNoFile(err error) error {
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

type storageFile struct {
	Calendars []CalendarAuth
}
