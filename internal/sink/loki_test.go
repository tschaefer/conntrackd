/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/common/model"
	slogloki "github.com/samber/slog-loki/v3"
	"github.com/stretchr/testify/assert"
)

func targetLoki_AddressInvalid(t *testing.T) {
	loki := &Loki{
		Enable:  true,
		Address: "://invalid-address",
	}
	handler, err := loki.TargetLoki(&slog.HandlerOptions{})
	assert.NotNil(t, err)
	assert.Nil(t, handler)
	assert.EqualError(t, err, "parse \"://invalid-address\": missing protocol scheme")
}

func targetLoki_AddressUnreachable(t *testing.T) {
	loki := &Loki{
		Enable:  true,
		Address: "http://example.invalid:1234",
	}
	handler, err := loki.TargetLoki(&slog.HandlerOptions{})
	assert.NotNil(t, err)
	assert.Nil(t, handler)
	assert.EqualError(t, err, "Get \"http://example.invalid:1234/ready\": dial tcp: lookup example.invalid on 127.0.0.53:53: no such host")
}

func targetLoki_AddressNotReady(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	loki := &Loki{
		Enable:  true,
		Address: ts.URL,
	}
	handler, err := loki.TargetLoki(&slog.HandlerOptions{})
	assert.NotNil(t, err)
	assert.Nil(t, handler)
	assert.EqualError(t, err, "404 Not Found")
}

func setLabels(t *testing.T) {
	loki := &Loki{
		Enable:  true,
		Address: "http://localhost:3100",
		Labels:  []string{"invalid-labels", "key=value"},
	}
	labels := loki.setLabels("hostname")

	assert.NotNil(t, labels)
	assert.Contains(t, labels.String(), "key=value")
	assert.NotContains(t, labels.String(), "invalid-labels")
	assert.Contains(t, labels.String(), "host=hostname")
}

func targetLoki(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	loki := &Loki{
		Enable:  true,
		Address: ts.URL,
	}
	handler, err := loki.TargetLoki(&slog.HandlerOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, handler)
	assert.IsType(t, &slogloki.LokiHandler{}, handler)
}

func testAttrsToMetadata(t *testing.T) {
	labels := map[string]string{
		"flow":      "1234567890",
		"prot":      "TCP",
		"src_addr":  "2003:cf:1716:7b64:da80:83ff:fecd:da51",
		"dst_addr":  "2a01:4f8:160:5372::2",
		"src_port":  "41756",
		"dst_port":  "443",
		"tcp_state": "SYN_SENT",
	}

	fields := map[string]string{
		"src_city":    "Garmisch-Partenkirchen",
		"src_country": "Germany",
		"src_lat":     "47.4906",
		"src_lon":     "11.1026",
		"dst_city":    "Falkenstein",
		"dst_country": "Germany",
		"dst_lat":     "50.4777",
		"dst_lon":     "12.3649",
	}

	addSource := false
	replaceAttr := func(groups []string, a slog.Attr) slog.Attr { return a }
	loggerAttrs := []slog.Attr{}
	record := slog.Record{
		Message: "Test log message",
	}
	for k, v := range labels {
		record.AddAttrs(slog.String(k, v))
	}
	for k, v := range fields {
		record.AddAttrs(slog.String(k, v))
	}

	recordLabels := attrsToMetadata(addSource, replaceAttr, loggerAttrs, []string{}, &record)
	assert.NotNil(t, recordLabels)
	assert.IsType(t, recordLabels, model.LabelSet{})
	for k, v := range labels {
		assert.Contains(t, recordLabels.String(), k+"=\""+v+"\"")
	}
	for k, v := range fields {
		assert.NotContains(t, recordLabels.String(), k+"=\""+v+"\"")
	}
}

func TestSinkTargetLoki(t *testing.T) {
	t.Run("TargetLoki returns error if address invalid", targetLoki_AddressInvalid)
	t.Run("TargetLoki returns error if address unreachable", targetLoki_AddressUnreachable)
	t.Run("TargetLoki returns error if address is not ready", targetLoki_AddressNotReady)
	t.Run("TargetLoki returns valid handler if address is reachable and ready", targetLoki)
	t.Run("setLabels returns only valid labels", setLabels)
	t.Run("attrsToMetadata returns valid labels", testAttrsToMetadata)
}
