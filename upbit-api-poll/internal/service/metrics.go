package service

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PrometheusService struct {
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

type PrometheusTimer struct {
	timer prometheus.Timer
}

func (t *PrometheusTimer) ObserveDuration() {
	t.timer.ObserveDuration()
}

func NewPrometheusService() *PrometheusService {
	return &PrometheusService{
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}

func (p *PrometheusService) IncrementCounter(name string, labels ...string) {
	counter := p.getOrCreateCounter(name, getLabelNames(labels))
	if len(labels) == 0 {
		counter.WithLabelValues().Inc()
	} else {
		counter.WithLabelValues(getLabelValues(labels)...).Inc()
	}
}

func (p *PrometheusService) SetGauge(name string, value float64, labels ...string) {
	gauge := p.getOrCreateGauge(name, getLabelNames(labels))
	if len(labels) == 0 {
		gauge.WithLabelValues().Set(value)
	} else {
		gauge.WithLabelValues(getLabelValues(labels)...).Set(value)
	}
}

func (p *PrometheusService) IncrementGauge(name string, labels ...string) {
	gauge := p.getOrCreateGauge(name, getLabelNames(labels))
	if len(labels) == 0 {
		gauge.WithLabelValues().Inc()
	} else {
		gauge.WithLabelValues(getLabelValues(labels)...).Inc()
	}
}

func (p *PrometheusService) DecrementGauge(name string, labels ...string) {
	gauge := p.getOrCreateGauge(name, getLabelNames(labels))
	if len(labels) == 0 {
		gauge.WithLabelValues().Dec()
	} else {
		gauge.WithLabelValues(getLabelValues(labels)...).Dec()
	}
}

func (p *PrometheusService) ObserveHistogram(name string, value float64, labels ...string) {
	histogram := p.getOrCreateHistogram(name, getLabelNames(labels))
	if len(labels) == 0 {
		histogram.WithLabelValues().Observe(value)
	} else {
		histogram.WithLabelValues(getLabelValues(labels)...).Observe(value)
	}
}

func (p *PrometheusService) StartTimer(name string, labels ...string) *PrometheusTimer {
	histogram := p.getOrCreateHistogram(name, getLabelNames(labels))
	var timer prometheus.Timer
	if len(labels) == 0 {
		timer = *prometheus.NewTimer(histogram.WithLabelValues())
	} else {
		timer = *prometheus.NewTimer(histogram.WithLabelValues(getLabelValues(labels)...))
	}
	return &PrometheusTimer{timer: timer}
}

func (p *PrometheusService) getOrCreateCounter(
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

func (p *PrometheusService) getOrCreateGauge(
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

func (p *PrometheusService) getOrCreateHistogram(
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

func (p *PrometheusService) StartMetricsServer(port string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":"+port, nil)
}

func (p *PrometheusService) GetHandler() http.Handler {
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
