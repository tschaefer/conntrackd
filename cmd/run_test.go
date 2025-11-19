package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStringSliceFlag_Valid(t *testing.T) {
	assert.NoError(t, validateStringSliceFlag("filter.include.types", []string{"NEW", "UPDATE"}, validEventTypes))
	assert.NoError(t, validateStringSliceFlag("filter.include.protocols", []string{"TCP"}, validProtocols))
	assert.NoError(t, validateStringSliceFlag("filter.exclude.addresses", []string{"127.0.0.1", "::1"}, []string{}))
}

func TestValidateStringSliceFlag_Invalid(t *testing.T) {
	assert.Error(t, validateStringSliceFlag("filter.include.types", []string{"BAD"}, validEventTypes))
	assert.Error(t, validateStringSliceFlag("filter.exclude.addresses", []string{"not-an-ip"}, []string{}))
}

func TestValidateStringFlag_LogLevelsAndFormats(t *testing.T) {
	assert.NoError(t, validateStringFlag("service.log.level", "debug", validLogLevels))
	assert.NoError(t, validateStringFlag("service.log.format", "json", validLogFormats))

	assert.Error(t, validateStringFlag("service.log.level", "verbose", validLogLevels))
	assert.Error(t, validateStringFlag("service.log.format", "xml", validLogFormats))
}

func TestValidateStringFlag_SyslogAddress_Valid(t *testing.T) {
	valids := []string{
		"udp://localhost:514",
		"tcp://127.0.0.1:514",
		"unix:///var/run/syslog.sock",
		"unixgram:///var/run/syslog.sock",
		"unixpacket:///var/run/syslog.sock",
	}
	for _, v := range valids {
		assert.NoErrorf(t, validateStringFlag("sink.syslog.address", v, []string{}), "valid syslog address %q should not error", v)
	}
}

func TestValidateStringFlag_SyslogAddress_Invalid(t *testing.T) {
	assert.Error(t, validateStringFlag("sink.syslog.address", "http://localhost:514", []string{}))
	assert.Error(t, validateStringFlag("sink.syslog.address", "tcp:///nohost", []string{}))
	assert.Error(t, validateStringFlag("sink.syslog.address", "unix://", []string{}))
}

func TestValidateStringFlag_LokiAddress_Valid(t *testing.T) {
	assert.NoError(t, validateStringFlag("sink.loki.address", "http://localhost:3100", []string{}))
	assert.NoError(t, validateStringFlag("sink.loki.address", "https://example.com", []string{}))
}

func TestValidateStringFlag_LokiAddress_Invalid(t *testing.T) {
	assert.Error(t, validateStringFlag("sink.loki.address", "tcp://localhost:3100", []string{}))
	assert.Error(t, validateStringFlag("sink.loki.address", "http:///path", []string{}))
}

func TestValidSlicesAreExplicit(t *testing.T) {
	expectedEvents := []string{"NEW", "UPDATE", "DESTROY"}
	assert.Equal(t, expectedEvents, validEventTypes, "validEventTypes mismatch")

	expectedProtocols := []string{"TCP", "UDP"}
	assert.Equal(t, expectedProtocols, validProtocols, "validProtocols mismatch")

	expectedDest := []string{"PUBLIC", "PRIVATE", "LOCAL", "MULTICAST"}
	assert.Equal(t, expectedDest, validDestinations, "validDestinations mismatch")
}
