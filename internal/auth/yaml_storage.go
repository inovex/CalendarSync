package auth

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	if strings.HasPrefix(y.StoragePath, "~"+string(filepath.Separator)) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		y.StoragePath = filepath.Join(homeDir, y.StoragePath[2:])
	}
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
	y.CachedAuth = cals

	return true, nil
}

func (y *YamlStorage) ReadCalendarAuth(calendarID string) (*CalendarAuth, error) {
	var calendars []CalendarAuth
	// if we already decrypted the file, read from memory
	if len(y.CachedAuth) > 0 {
		log.Debug("Loading Auth Data from memory")
		calendars = y.CachedAuth
	} else {
		log.Debug("Loading Auth Data from file")
		file, err := y.readAndParseFile()
		if err != nil {
			return nil, ignoreNoFile(err)
		}
		// Load data from file into cache
		y.CachedAuth = file.Calendars
		// use cached data from now on
		calendars = y.CachedAuth
	}

	for _, cal := range calendars {
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
	defer func() {
		err = file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

	writer = file

	// if encryption is enabled encrypt data
	if y.StorageEncryptionKey != "" {
		eFile := NewEncryptedFile(file, y.StorageEncryptionKey)
		defer func() {
			err = eFile.Close()
			if err != nil {
				log.Fatal(err)
			}
		}()
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
	defer func() {
		err = file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}()

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
