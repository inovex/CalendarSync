package config_test

import (
	"testing"

	"github.com/inovex/CalendarSync/internal/config"
	"github.com/inovex/CalendarSync/internal/sync"
	"github.com/inovex/CalendarSync/internal/transformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (suite *ConfigTestSuite) TestLoadingTransformersFromFile() {
	sut, err := config.NewFromFile("../../testdata/testconfig.yaml")

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), sut)

	loadedTransformers := sync.TransformerFactory(sut.Transformations)
	require.Truef(suite.T(), len(loadedTransformers) >= 5, "there must be at least five transformers in the config file")

	keepAttendees := loadedTransformers[0].(*transformation.KeepAttendees)
	assert.NotNil(suite.T(), keepAttendees.Name())
	assert.Equal(suite.T(), true, keepAttendees.UseEmailAsDisplayName)

	keepDescription := loadedTransformers[1].(*transformation.KeepDescription)
	assert.NotNil(suite.T(), keepDescription.Name())

	keepTitle := loadedTransformers[2].(*transformation.KeepTitle)
	assert.NotNil(suite.T(), keepTitle.Name())

	prefixTitle := loadedTransformers[3].(*transformation.PrefixTitle)
	assert.NotNil(suite.T(), prefixTitle.Name())
	assert.Equal(suite.T(), "foobar", prefixTitle.Prefix)

	replaceTitle := loadedTransformers[4].(*transformation.ReplaceTitle)
	assert.NotNil(suite.T(), replaceTitle.Name())
	assert.Equal(suite.T(), "[Synchronisierter Termin]", replaceTitle.NewTitle)
}

func (suite *ConfigTestSuite) TestAuthStorageDefaultsFromFile() {
	sut, err := config.NewFromFile("../../testdata/empty_testconfig.yaml")

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), sut)

	assert.Equal(suite.T(), "yaml", sut.Auth.StorageMode)
	assert.Equal(suite.T(), "./auth-storage.yaml", sut.Auth.Config["path"])
}

func (suite *ConfigTestSuite) TestCustomAuthStorageFromFile() {
	sut, err := config.NewFromFile("../../testdata/custom_auth_storage.yaml")

	assert.Nil(suite.T(), err)
	assert.NotNil(suite.T(), sut)

	assert.Equal(suite.T(), "custom", sut.Auth.StorageMode)
	assert.Equal(suite.T(), "./auth-storage.custom", sut.Auth.Config["path"])
}
