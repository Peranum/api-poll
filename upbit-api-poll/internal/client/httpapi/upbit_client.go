package httpapi

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/service"
	"github.com/google/uuid"
)

const (
	ResponseTimeout        = 1 * time.Second
	InitialCacheBufferSize = 512
	MaxResponseBodySize    = 2 * 1024 * 1024 // 2MB, a safe upper limit for API responses
	UserAgent              = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"

	// Metric names
	MetricUpbitRequestDuration    = "upbit_client_request_duration"
	MetricUpbitRequestsTotal      = "upbit_client_requests_total"
	MetricUpbitErrorsTotal        = "upbit_client_errors_total"
	MetricUpbitResponsesTotal     = "upbit_client_responses_total"
	MetricUpbitCacheHitsTotal     = "upbit_client_cache_hits_total"
	MetricUpbitSuccessfulRequests = "upbit_client_successful_requests_total"
	MetricUpbitRateLimitedTotal   = "upbit_client_rate_limited_total"
)

var (
	ErrRequestTimeout = errors.New("request timeout")
	ErrRateLimited    = errors.New("rate limited")
)

type UpbitClient struct {
	proxyAddr  string
	httpClient *http.Client

	url                  string
	maxRetries           int
	retryDelay           time.Duration
	retryDelayMultiplier float64

	metrics service.MetricsService

	logger *slog.Logger
}

func NewUpbitClient(
	cfg config.UpbitClient,
	proxy *entity.Proxy,
	logger *slog.Logger,
	metrics service.MetricsService,
) (*UpbitClient, error) {
	transport := &http.Transport{
		MaxConnsPerHost:     100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     time.Second * 15,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		ForceAttemptHTTP2: true,
	}

	proxyAddr := "direct"
	if proxy != nil && proxy.String() != "" {
		proxyURL, err := url.Parse(proxy.String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
		}

		proxyAddr = proxy.String()
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	httpClient := &http.Client{
		Transport: transport,
	}

	client := &UpbitClient{
		url:                  cfg.UpbitAnnouncementsURL,
		maxRetries:           cfg.MaxRetries,
		retryDelay:           cfg.RetryDelay,
		retryDelayMultiplier: cfg.RetryDelayMultiplier,
		proxyAddr:            proxyAddr,
		httpClient:           httpClient,
		metrics:              metrics,
		logger:               logger,
	}

	return client, nil
}

func (u *UpbitClient) Request(ctx context.Context) (entity.UpbitResponse, error) {
	timer := u.metrics.StartTimer(MetricUpbitRequestDuration, "proxy", u.proxyAddr)
	defer timer.ObserveDuration()

	u.metrics.IncrementCounter(MetricUpbitRequestsTotal, "proxy", u.proxyAddr)

	ctx, cancel := context.WithTimeout(ctx, ResponseTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.url, nil)
	if err != nil {
		u.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			u.proxyAddr,
			"type",
			"request_creation",
		)

		return entity.UpbitResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept-Language", "en-KR, en;q=1, ru-RU;q=0.1")
	req.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")
	req.Header.Set("Origin", fmt.Sprintf("https://%s.com", uuid.New().String()))

	resp, err := u.httpClient.Do(req)
	if err != nil {
		u.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			u.proxyAddr,
			"type",
			"request_failed",
		)

		return entity.UpbitResponse{}, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		u.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			u.proxyAddr,
			"type",
			"body_read",
		)

		return entity.UpbitResponse{}, err
	}

	u.metrics.IncrementCounter(MetricUpbitResponsesTotal,
		"proxy", u.proxyAddr,
		"status_code", strconv.Itoa(resp.StatusCode),
		"status_class", u.getStatusClass(resp.StatusCode))

	upbitResp := entity.NewUpbitResponse(resp.StatusCode, resp.Header, body, u.proxyAddr)

	// Handle rate limiting
	if resp.StatusCode == http.StatusTooManyRequests {
		u.metrics.IncrementCounter(MetricUpbitRateLimitedTotal, "proxy", u.proxyAddr)
		u.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			u.proxyAddr,
			"type",
			"rate_limited",
		)

		u.logger.Info(
			"News fetched",
			"Cf-Cache-Status", upbitResp.Headers.Get("Cf-Cache-Status"),
			"Cf-Ray", upbitResp.Headers.Get("Cf-Ray"),
			"X-Request-ID", upbitResp.Headers.Get("X-Request-ID"),
			"X-Runtime", upbitResp.Headers.Get("X-Runtime"),
			"Retry-After", upbitResp.Headers.Get("Retry-After"),
			"proxy", u.proxyAddr,
		)

		return upbitResp, ErrRateLimited
	}

	// Handle successful response
	if resp.StatusCode == http.StatusOK {
		u.metrics.IncrementCounter(MetricUpbitSuccessfulRequests, "proxy", u.proxyAddr)

		return upbitResp, nil
	}

	return upbitResp, nil
}

func (u *UpbitClient) getStatusClass(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "2xx"
	case statusCode >= 300 && statusCode < 400:
		return "3xx"
	case statusCode >= 400 && statusCode < 500:
		return "4xx"
	case statusCode >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}
