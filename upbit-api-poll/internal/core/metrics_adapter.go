package core

import (
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/service"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/httptools"
)

type metricsAdapter struct {
	prometheus *service.PrometheusService
}

func (a *metricsAdapter) IncrementCounter(name string, labels ...string) {
	a.prometheus.IncrementCounter(name, labels...)
}

func (a *metricsAdapter) SetGauge(name string, value float64, labels ...string) {
	a.prometheus.SetGauge(name, value, labels...)
}

func (a *metricsAdapter) IncrementGauge(name string, labels ...string) {
	a.prometheus.IncrementGauge(name, labels...)
}

func (a *metricsAdapter) DecrementGauge(name string, labels ...string) {
	a.prometheus.DecrementGauge(name, labels...)
}

func (a *metricsAdapter) ObserveHistogram(name string, value float64, labels ...string) {
	a.prometheus.ObserveHistogram(name, value, labels...)
}

func (a *metricsAdapter) StartTimer(name string, labels ...string) httptools.Timer {
	return a.prometheus.StartTimer(name, labels...)
}
