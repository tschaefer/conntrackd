/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	kitlog "github.com/go-kit/log"
	kitlevel "github.com/go-kit/log/level"
	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/loki-client-go/pkg/labelutil"
	"github.com/prometheus/common/model"
	slogcommon "github.com/samber/slog-common"
	slogloki "github.com/samber/slog-loki/v3"
	"github.com/tschaefer/conntrackd/internal/logger"
)

const (
	readyPath = "/ready"
	pushPath  = "/loki/api/v1/push"
)

// Loki represents Loki logging sink.
type Loki struct {
	Enable  bool
	Address string
	Labels  []string
}

// Supported Loki protocols.
var LokiProtocols = []string{"http", "https"}

// Attributes to be used as labels in Loki payload.
var labelAttrs = []string{
	"flow", "type", "prot",
	"src_addr", "src_port", "dst_addr", "dst_port",
	"tcp_state",
}

// TargetLoki creates a sink target for Loki.
func (l *Loki) TargetLoki(options *slog.HandlerOptions) (slog.Handler, error) {
	url, err := url.Parse(l.Address)
	if err != nil {
		return nil, err
	}

	if err := l.isReady(*url); err != nil {
		return nil, err
	}

	url.Path = url.Path + pushPath
	config, err := loki.NewDefaultConfig(url.String())
	if err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	config.ExternalLabels = l.setLabels(hostname)

	klogger := l.createLogger()
	client, err := loki.NewWithLogger(config, klogger)
	if err != nil {
		return nil, err
	}

	o := &slogloki.Option{
		Client:                    client,
		Level:                     options.Level,
		HandleRecordsWithMetadata: true,
		Converter:                 attrsToMetadata,
	}
	return o.NewLokiHandler(), nil
}

// isReady checks if Loki server is ready to accept requests.
func (l *Loki) isReady(url url.URL) error {
	url.Path = url.Path + readyPath

	response, err := http.Get(url.String())
	if err != nil {
		return err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode != http.StatusOK {
		return errors.New(response.Status)
	}

	return nil
}

// setLabels sets external labels for Loki client.
func (l *Loki) setLabels(hostname string) labelutil.LabelSet {
	labels := labelutil.LabelSet{
		LabelSet: model.LabelSet{
			model.LabelName("service_name"): model.LabelValue("conntrackd"),
			model.LabelName("host"):         model.LabelValue(hostname),
		},
	}

	if len(l.Labels) == 0 {
		return labels
	}

	for _, label := range l.Labels {
		if !strings.Contains(label, "=") {
			continue
		}
		parts := strings.SplitN(label, "=", 2)
		key := parts[0]
		value := parts[1]
		labels.LabelSet[model.LabelName(key)] = model.LabelValue(value)
	}

	return labels
}

// createLogger creates a go-kit logger for Loki client.
func (l *Loki) createLogger() kitlog.Logger {
	level := logger.Level().String()
	klevel := kitlevel.ParseDefault(level, kitlevel.InfoValue())

	klogger := kitlog.NewJSONLogger(kitlog.NewSyncWriter(os.Stderr))
	klogger = kitlevel.NewFilter(klogger, kitlevel.Allow(klevel))
	klogger = kitlog.With(klogger, "time", kitlog.DefaultTimestamp, "sink", "loki")

	return klogger
}

// attrsToMetadata converts slog attributes to Loki metadata labels.
func attrsToMetadata(addSource bool, replaceAttr func(groups []string, a slog.Attr) slog.Attr, loggerAttr []slog.Attr, groups []string, record *slog.Record) model.LabelSet {
	attrs := slogcommon.AppendRecordAttrsToAttrs(loggerAttr, groups, record)

	newRecord := slog.NewRecord(record.Time, record.Level, record.Message, record.PC)
	for _, attr := range attrs {
		if slices.Contains(labelAttrs, attr.Key) {
			newRecord.AddAttrs(attr)
		}
	}

	return slogloki.DefaultConverter(addSource, replaceAttr, loggerAttr, groups, &newRecord)
}
