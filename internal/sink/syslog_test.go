/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"log/slog"
	"testing"

	slogsyslog "github.com/samber/slog-syslog/v2"
	"github.com/stretchr/testify/assert"
)

func targetSyslog_AddressValid(t *testing.T) {
	syslog := &Syslog{
		Enable:  true,
		Address: "udp://localhost:514",
	}
	handler, err := syslog.TargetSyslog(&slog.HandlerOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, handler)
	assert.IsType(t, &slogsyslog.SyslogHandler{}, handler)
}

func targetSyslog_AddressInvalid(t *testing.T) {
	type data struct {
		address string
		errMsg  string
	}

	datas := []data{
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

	for _, d := range datas {
		syslog := &Syslog{
			Enable:  true,
			Address: d.address,
		}
		handler, err := syslog.TargetSyslog(&slog.HandlerOptions{})
		assert.NotNil(t, err)
		assert.Nil(t, handler)
		assert.EqualError(t, err, d.errMsg)
	}
}

func TestSinkTargetSyslog(t *testing.T) {
	t.Run("TargetSyslog returns valid writer if address valid", targetSyslog_AddressValid)
	t.Run("TargetSyslog returns error if address invalid", targetSyslog_AddressInvalid)
}
