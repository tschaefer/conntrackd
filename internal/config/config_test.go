/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func initConfigSucceedsIfConfigFileIsAvailable(t *testing.T) {
	content := `
log:
  level: debug
geoip:
  database: /path/to/db.mmdb
filter:
  - "drop any"
sink:
  stream:
    enable: true
    writer: stdout
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(content))
	assert.NoError(t, err)
	defer tmpfile.Close()

	viper.Reset()

	err = InitConfig(tmpfile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "debug", viper.GetString("log.level"))
	assert.Equal(t, "/path/to/db.mmdb", viper.GetString("geoip.database"))
	assert.Equal(t, []any{"drop any"}, viper.Get("filter"))
	assert.Equal(t, true, viper.GetBool("sink.stream.enable"))
	assert.Equal(t, "stdout", viper.GetString("sink.stream.writer"))
}

func initConfigSucceedsIfNoConfigFileIsAvailable(t *testing.T) {
	viper.Reset()

	err := InitConfig("")
	assert.NoError(t, err)
}

func TestInitConfigReturnsErrorIfConfigFileIsNotFound(t *testing.T) {
	viper.Reset()

	err := InitConfig("/nonexistent/config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func initConfigReturnsErrorIfConfigFileHasInvalidYAML(t *testing.T) {
	content := `
invalid yaml content:
  - this is not valid
    because: indentation is wrong
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(content))
	assert.NoError(t, err)
	defer tmpfile.Close()

	viper.Reset()

	err = InitConfig(tmpfile.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reading config file")
}

func initConfigSucceedsIfEnvironmentVariableOverridesSettings(t *testing.T) {
	content := `
log:
  level: info
sink:
  stream:
    enable: true
    writer: stdout
`
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(content))
	assert.NoError(t, err)
	tmpfile.Close()

	viper.Reset()

	os.Setenv("CONNTRACKD_SINK_STREAM_WRITER", "discard")
	os.Setenv("CONNTRACKD_LOG_LEVEL", "debug")
	defer os.Unsetenv("CONNTRACKD_SINK_STREAM_WRITER")
	defer os.Unsetenv("CONNTRACKD_LOG_LEVEL")

	err = InitConfig(tmpfile.Name())
	assert.NoError(t, err)

	assert.Equal(t, "discard", viper.GetString("sink.stream.writer"))
	assert.Equal(t, "debug", viper.GetString("log.level"))
}

func TestConfig(t *testing.T) {
	t.Run("config.InitConfig succeeds if config file is available", initConfigSucceedsIfConfigFileIsAvailable)
	t.Run("config.InitConfig succeeds if no config file is available", initConfigSucceedsIfNoConfigFileIsAvailable)
	t.Run("config.InitConfig returns error if config file is not found", TestInitConfigReturnsErrorIfConfigFileIsNotFound)
	t.Run("config.InitConfig returns error if config file has invalid YAML", initConfigReturnsErrorIfConfigFileHasInvalidYAML)
	t.Run("config.InitConfig succeeds if environment variable overrides settings", initConfigSucceedsIfEnvironmentVariableOverridesSettings)
}
