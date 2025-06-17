package service

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsService interface {
	// Counters
	IncrementCounter(name string, labels ...string)

	// Gauges
	SetGauge(name string, value float64, labels ...string)
	IncrementGauge(name string, labels ...string)
	DecrementGauge(name string, labels ...string)

	// Histograms
	ObserveHistogram(name string, value float64, labels ...string)
	StartTimer(name string, labels ...string) Timer

	// Server
	StartMetricsServer(port string) error
	GetHandler() http.Handler
}

type Timer interface {
	ObserveDuration()
}

type prometheusService struct {
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

type prometheusTimer struct {
	timer prometheus.Timer
}

func (t *prometheusTimer) ObserveDuration() {
	t.timer.ObserveDuration()
}

func NewPrometheusService() MetricsService {
	return &prometheusService{
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}

func (p *prometheusService) IncrementCounter(name string, labels ...string) {
	counter := p.getOrCreateCounter(name, getLabelNames(labels))
	if len(labels) == 0 {
		counter.WithLabelValues().Inc()
	} else {
		counter.WithLabelValues(getLabelValues(labels)...).Inc()
	}
}

func (p *prometheusService) SetGauge(name string, value float64, labels ...string) {
	gauge := p.getOrCreateGauge(name, getLabelNames(labels))
	if len(labels) == 0 {
		gauge.WithLabelValues().Set(value)
	} else {
		gauge.WithLabelValues(getLabelValues(labels)...).Set(value)
	}
}

func (p *prometheusService) IncrementGauge(name string, labels ...string) {
	gauge := p.getOrCreateGauge(name, getLabelNames(labels))
	if len(labels) == 0 {
		gauge.WithLabelValues().Inc()
	} else {
		gauge.WithLabelValues(getLabelValues(labels)...).Inc()
	}
}

func (p *prometheusService) DecrementGauge(name string, labels ...string) {
	gauge := p.getOrCreateGauge(name, getLabelNames(labels))
	if len(labels) == 0 {
		gauge.WithLabelValues().Dec()
	} else {
		gauge.WithLabelValues(getLabelValues(labels)...).Dec()
	}
}

func (p *prometheusService) ObserveHistogram(name string, value float64, labels ...string) {
	histogram := p.getOrCreateHistogram(name, getLabelNames(labels))
	if len(labels) == 0 {
		histogram.WithLabelValues().Observe(value)
	} else {
		histogram.WithLabelValues(getLabelValues(labels)...).Observe(value)
	}
}

func (p *prometheusService) StartTimer(name string, labels ...string) Timer {
	histogram := p.getOrCreateHistogram(name, getLabelNames(labels))
	var timer prometheus.Timer
	if len(labels) == 0 {
		timer = *prometheus.NewTimer(histogram.WithLabelValues())
	} else {
		timer = *prometheus.NewTimer(histogram.WithLabelValues(getLabelValues(labels)...))
	}
	return &prometheusTimer{timer: timer}
}

func (p *prometheusService) getOrCreateCounter(
	name string,
	labelNames []string,
) *prometheus.CounterVec {
	if counter, exists := p.counters[name]; exists {
		return counter
	}

	counter := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: name + " counter",
		},
		labelNames,
	)
	p.counters[name] = counter
	return counter
}

func (p *prometheusService) getOrCreateGauge(
	name string,
	labelNames []string,
) *prometheus.GaugeVec {
	if gauge, exists := p.gauges[name]; exists {
		return gauge
	}

	gauge := promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: name + " gauge",
		},
		labelNames,
	)
	p.gauges[name] = gauge
	return gauge
}

func (p *prometheusService) getOrCreateHistogram(
	name string,
	labelNames []string,
) *prometheus.HistogramVec {
	if histogram, exists := p.histograms[name]; exists {
		return histogram
	}

	histogram := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    name + " histogram",
			Buckets: prometheus.DefBuckets,
		},
		labelNames,
	)
	p.histograms[name] = histogram
	return histogram
}

func (p *prometheusService) StartMetricsServer(port string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":"+port, nil)
}

func (p *prometheusService) GetHandler() http.Handler {
	return promhttp.Handler()
}

// Helper functions to extract label names and values from alternating key-value pairs
func getLabelNames(labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	labelNames := make([]string, 0, len(labels)/2)
	for i := 0; i < len(labels); i += 2 {
		if i < len(labels) {
			labelNames = append(labelNames, labels[i])
		}
	}
	return labelNames
}

func getLabelValues(labels []string) []string {
	if len(labels) == 0 {
		return nil
	}
	labelValues := make([]string, 0, len(labels)/2)
	for i := 1; i < len(labels); i += 2 {
		if i < len(labels) {
			labelValues = append(labelValues, labels[i])
		}
	}
	return labelValues
}
