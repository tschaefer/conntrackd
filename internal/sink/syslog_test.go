/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"log/slog"
	"testing"

	slogsyslog "github.com/samber/slog-syslog/v2"
	"github.com/stretchr/testify/assert"
)

func targetSyslogReturnsHandlerIfAddressIsValid(t *testing.T) {
	syslog := &Syslog{
		Enable:  true,
		Address: "udp://localhost:514",
	}
	handler, err := syslog.TargetSyslog(&slog.HandlerOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, handler)
	assert.IsType(t, &slogsyslog.SyslogHandler{}, handler)
}

func targetSyslogReturnsErrorIfAddressIsInvalid(t *testing.T) {
	cases := []struct {
		address string
		errMsg  string
	}{
		{
			address: "unix:///dev/notfound",
			errMsg:  "dial unix /dev/notfound: connect: no such file or directory",
		},
		{
			address: "://invalid-address",
			errMsg:  "parse \"://invalid-address\": missing protocol scheme",
		},
		{
			address: "invalid-protocol://localhost:514",
			errMsg:  "dial invalid-protocol: unknown network invalid-protocol",
		},
	}

	for _, tc := range cases {
		syslog := &Syslog{
			Enable:  true,
			Address: tc.address,
		}
		handler, err := syslog.TargetSyslog(&slog.HandlerOptions{})
		assert.NotNil(t, err)
		assert.Nil(t, handler)
		assert.EqualError(t, err, tc.errMsg)
	}
}

func TestSinkTargetSyslog(t *testing.T) {
	t.Run("syslog.TargetSyslog returns handler if address is valid", targetSyslogReturnsHandlerIfAddressIsValid)
	t.Run("syslog.TargetSyslog returns error if address is invalid", targetSyslogReturnsErrorIfAddressIsInvalid)
}
