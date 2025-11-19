/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package sink

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	klog "github.com/go-kit/log"
	"github.com/grafana/loki-client-go/loki"
	"github.com/grafana/loki-client-go/pkg/labelutil"
	"github.com/prometheus/common/model"
	slogloki "github.com/samber/slog-loki/v3"
	"github.com/tschaefer/conntrackd/internal/logger"
)

type Loki struct {
	Enable  bool
	Address string
	Labels  []string
}

const (
	readyPath = "/ready"
	pushPath  = "/loki/api/v1/push"
)

func (l *Loki) isReady() error {
	uri, err := url.Parse(l.Address)
	if err != nil {
		return err
	}
	uri.Path = uri.Path + readyPath

	response, err := http.Get(uri.String())
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

func (l *Loki) TargetLoki(options *slog.HandlerOptions) (slog.Handler, error) {
	slog.Debug("Initializing Grafana Loki sink.", "data", l)

	if err := l.isReady(); err != nil {
		return nil, err
	}

	uri, err := url.Parse(l.Address)
	if err != nil {
		return nil, err
	}
	uri.Path = uri.Path + pushPath

	config, err := loki.NewDefaultConfig(uri.String())
	if err != nil {
		return nil, err
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	config.ExternalLabels = labelutil.LabelSet{
		LabelSet: model.LabelSet{
			model.LabelName("service_name"): model.LabelValue("conntrackd"),
			model.LabelName("host"):         model.LabelValue(hostname),
		},
	}

	if len(l.Labels) > 0 {
		for _, label := range l.Labels {
			if !strings.Contains(label, "=") {
				continue
			}
			parts := strings.SplitN(label, "=", 2)
			key := parts[0]
			value := parts[1]
			config.ExternalLabels.LabelSet[model.LabelName(key)] = model.LabelValue(value)
		}
	}

	sw := klog.NewSyncWriter(os.Stderr)
	var klogger klog.Logger
	switch logger.Format() {
	case "json":
		klogger = klog.NewJSONLogger(sw)
	case "text":
		fallthrough
	default:
		klogger = klog.NewLogfmtLogger(sw)
	}
	klogger = klog.With(klogger, "time", klog.DefaultTimestamp, "sink", "loki")

	client, err := loki.NewWithLogger(config, klogger)
	if err != nil {
		return nil, err
	}

	o := &slogloki.Option{
		Client: client,
		Level:  options.Level,
	}
	return o.NewLokiHandler(), nil
}
