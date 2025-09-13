package httptools

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"time"
)

var (
	ErrTargetRPSRequired         = errors.New("targetRPS is required")
	ErrSingleProxyMaxRPSRequired = errors.New("singleProxyMaxRPS is required")
	ErrInitClientFnRequired      = errors.New("initClientFn is required")
	ErrProxiesRequired           = errors.New("proxies are required")
)

type ProxyRotatingPollerBuilder struct {
	url                 string
	targetRPS           float64
	singleProxyMaxRPS   float64
	proxies             []Proxy
	initProxyClientFn   func(proxy Proxy) (Client, error)
	clientRetriesConfig ClientRetriesConfig
	logger              *slog.Logger
	workSchedule        WorkSchedule
	notifier            Notifier
	metrics             Metrics
}

func NewProxyRotatingPollerBuilder() *ProxyRotatingPollerBuilder {
	return &ProxyRotatingPollerBuilder{
		logger:   slog.New(slog.DiscardHandler),
		notifier: noopNotifier{},
		metrics:  noopMetrics{},
		initProxyClientFn: func(proxy Proxy) (Client, error) {
			return NewClientHTTP2(ClientConfig{
				Proxy:   &proxy,
				Metrics: noopMetrics{},
				Logger:  slog.New(slog.DiscardHandler),
				ClientRetriesConfig: ClientRetriesConfig{
					MaxRetries:           3,
					RetryDelay:           1 * time.Second,
					RetryDelayMultiplier: 2,
				},
			})
		},
		workSchedule: noopWorkSchedule{},
	}
}

func (b *ProxyRotatingPollerBuilder) Build() (*ProxyRotatingPoller, error) {
	if _, err := url.Parse(b.url); err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	if b.targetRPS <= 0 {
		return nil, ErrTargetRPSRequired
	}

	if b.singleProxyMaxRPS <= 0 {
		return nil, ErrSingleProxyMaxRPSRequired
	}

	if b.initProxyClientFn == nil {
		return nil, ErrInitClientFnRequired
	}

	if len(b.proxies) == 0 {
		return nil, ErrProxiesRequired
	}

	clients := make([]Client, 0, len(b.proxies))
	for _, proxy := range b.proxies {
		client, err := b.initProxyClientFn(proxy)
		if err != nil {
			return nil, err
		}

		clients = append(clients, client)
	}

	pollingRatePerProxy := time.Duration(float64(time.Second) / b.singleProxyMaxRPS)
	clientsByLocation := NewClientsQueuePool(clients, pollingRatePerProxy)

	minExpectedProxiesCount := int(math.Ceil(b.targetRPS / b.singleProxyMaxRPS))
	if clientsByLocation.Len() < minExpectedProxiesCount {
		return nil, fmt.Errorf(
			"expected at least %d proxies, got %d",
			minExpectedProxiesCount,
			clientsByLocation.Len(),
		)
	}

	return &ProxyRotatingPoller{
		url:               b.url,
		targetRPS:         b.targetRPS,
		singleProxyMaxRPS: b.singleProxyMaxRPS,
		clientsByLocation: clientsByLocation,
		totalClientsCount: len(clients),
		workSchedule:      b.workSchedule,
		logger:            b.logger,
		notifier:          b.notifier,
		metrics:           b.metrics,
	}, nil
}

func (b *ProxyRotatingPollerBuilder) WithURL(url string) *ProxyRotatingPollerBuilder {
	b.url = url
	return b
}

func (b *ProxyRotatingPollerBuilder) WithTargetRPS(targetRPS float64) *ProxyRotatingPollerBuilder {
	b.targetRPS = targetRPS
	return b
}

func (b *ProxyRotatingPollerBuilder) WithSingleProxyMaxRPS(
	singleProxyMaxRPS float64,
) *ProxyRotatingPollerBuilder {
	b.singleProxyMaxRPS = singleProxyMaxRPS
	return b
}

func (b *ProxyRotatingPollerBuilder) WithProxies(proxies ...Proxy) *ProxyRotatingPollerBuilder {
	b.proxies = proxies
	return b
}

func (b *ProxyRotatingPollerBuilder) WithInitProxyClientFn(
	initProxyClientFn func(proxy Proxy) (Client, error),
) *ProxyRotatingPollerBuilder {
	b.initProxyClientFn = initProxyClientFn
	return b
}

func (b *ProxyRotatingPollerBuilder) WithWorkSchedule(
	workSchedule WorkSchedule,
) *ProxyRotatingPollerBuilder {
	b.workSchedule = workSchedule
	return b
}

func (b *ProxyRotatingPollerBuilder) WithLogger(logger *slog.Logger) *ProxyRotatingPollerBuilder {
	b.logger = logger
	return b
}

func (b *ProxyRotatingPollerBuilder) UseFastHTTP(
	config ...ClientConfig,
) *ProxyRotatingPollerBuilder {
	cfg := ClientConfig{
		Logger:  slog.New(slog.DiscardHandler),
		Metrics: noopMetrics{},
		ClientRetriesConfig: ClientRetriesConfig{
			MaxRetries:           3,
			RetryDelay:           1 * time.Second,
			RetryDelayMultiplier: 2,
		},
	}

	if len(config) > 0 {
		cfg = config[0]
	}

	b.initProxyClientFn = func(proxy Proxy) (Client, error) {
		return NewClientFastHTTP(ClientConfig{
			Proxy:               &proxy,
			Logger:              slog.New(slog.DiscardHandler),
			Metrics:             noopMetrics{},
			ClientRetriesConfig: cfg.ClientRetriesConfig,
		})
	}

	return b
}

func (b *ProxyRotatingPollerBuilder) UseHTTP2(config ...ClientConfig) *ProxyRotatingPollerBuilder {
	cfg := ClientConfig{
		Logger:  slog.New(slog.DiscardHandler),
		Metrics: noopMetrics{},
		ClientRetriesConfig: ClientRetriesConfig{
			MaxRetries:           3,
			RetryDelay:           1 * time.Second,
			RetryDelayMultiplier: 2,
		},
	}

	if len(config) > 0 {
		cfg = config[0]
	}

	b.initProxyClientFn = func(proxy Proxy) (Client, error) {
		return NewClientHTTP2(ClientConfig{
			Proxy:               &proxy,
			Logger:              slog.New(slog.DiscardHandler),
			Metrics:             noopMetrics{},
			ClientRetriesConfig: cfg.ClientRetriesConfig,
		})
	}

	return b
}

type noopNotifier struct{}

func (n noopNotifier) SendMessage(message string, args ...interface{}) {}

func (b *ProxyRotatingPollerBuilder) WithNotifier(notifier Notifier) *ProxyRotatingPollerBuilder {
	b.notifier = notifier
	return b
}

type noopMetrics struct{}

func (n noopMetrics) IncrementCounter(name string, labels ...string)                {}
func (n noopMetrics) SetGauge(name string, value float64, labels ...string)         {}
func (n noopMetrics) IncrementGauge(name string, labels ...string)                  {}
func (n noopMetrics) DecrementGauge(name string, labels ...string)                  {}
func (n noopMetrics) ObserveHistogram(name string, value float64, labels ...string) {}
func (n noopMetrics) StartTimer(name string, labels ...string) Timer {
	return &noopTimer{}
}

type noopTimer struct{}

func (n noopTimer) ObserveDuration() {}

func (b *ProxyRotatingPollerBuilder) WithMetrics(metrics Metrics) *ProxyRotatingPollerBuilder {
	b.metrics = metrics
	return b
}

type noopWorkSchedule struct{}

func (n noopWorkSchedule) WorkNow() bool {
	return true
}

func (n noopWorkSchedule) NextWorkSession() (time.Duration, error) {
	return 0, nil
}
