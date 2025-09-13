package httptools

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
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

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
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

var ErrRequestTimeout = errors.New("request timeout")

type ClientHTTP2 struct {
	ipAddress  string
	proxyAddr  string
	httpClient *http.Client

	url                  string
	maxRetries           int
	retryDelay           time.Duration
	retryDelayMultiplier float64

	metrics Metrics

	logger *slog.Logger
}

func NewClientHTTP2(
	config ...ClientConfig,
) (*ClientHTTP2, error) {
	cfg := ClientConfig{
		Logger:  slog.New(slog.DiscardHandler),
		Metrics: noopMetrics{},
		ClientRetriesConfig: ClientRetriesConfig{
			MaxRetries:           3,
			RetryDelay:           5 * time.Second,
			RetryDelayMultiplier: 2,
		},
	}

	if len(config) > 0 {
		cfg = config[0]
	}

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
	if cfg.Proxy != nil && cfg.Proxy.String() != "" {
		proxyURL, err := url.Parse(cfg.Proxy.String())
		if err != nil {
			return nil, fmt.Errorf("failed to parse proxy URL: %w", err)
		}

		proxyAddr = cfg.Proxy.String()
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	httpClient := &http.Client{
		Transport: transport,
	}

	host := "127.0.0.1"
	if cfg.Proxy != nil {
		host = cfg.Proxy.Host
	}

	client := &ClientHTTP2{
		logger:               cfg.Logger,
		metrics:              cfg.Metrics,
		maxRetries:           cfg.MaxRetries,
		retryDelay:           cfg.RetryDelay,
		retryDelayMultiplier: cfg.RetryDelayMultiplier,
		ipAddress:            host,
		proxyAddr:            proxyAddr,
		httpClient:           httpClient,
	}

	return client, nil
}

func (c *ClientHTTP2) Request(ctx context.Context, url string) (Response, error) {
	requestedAt := time.Now()

	timer := c.metrics.StartTimer(MetricUpbitRequestDuration, "proxy", c.proxyAddr)
	defer timer.ObserveDuration()

	c.metrics.IncrementCounter(MetricUpbitRequestsTotal, "proxy", c.proxyAddr)

	ctx, cancel := context.WithTimeout(ctx, ResponseTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			c.proxyAddr,
			"type",
			"request_creation",
		)

		return Response{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept-Language", "ko-KR, ko;q=1, en-US;q=0.1")
	req.Header.Set(
		"Cache-Control",
		"no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0",
	)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")
	// req.Header.Set("Origin", "https://binance.com")
	req.Header.Set("Priority", "u=0")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			c.proxyAddr,
			"type",
			"request_failed",
		)

		return Response{}, err
	}

	defer resp.Body.Close()

	body, err := c.decodeResponseBody(resp)
	if err != nil {
		c.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			c.proxyAddr,
			"type",
			"body_read",
		)

		return Response{}, err
	}

	c.metrics.IncrementCounter(MetricUpbitResponsesTotal,
		"proxy", c.proxyAddr,
		"status_code", strconv.Itoa(resp.StatusCode),
		"status_class", c.getStatusClass(resp.StatusCode))

	upbitResp := NewHTTPResponse(
		requestedAt,
		resp.StatusCode,
		resp.Header,
		body,
		c.proxyAddr,
		"http/2.0",
	)

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		c.metrics.IncrementCounter(MetricUpbitRateLimitedTotal, "proxy", c.proxyAddr)
		c.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			c.proxyAddr,
			"type",
			"rate_limited",
		)
	case http.StatusOK:
		c.metrics.IncrementCounter(MetricUpbitSuccessfulRequests, "proxy", c.proxyAddr)
	}

	return upbitResp, nil
}

func (c *ClientHTTP2) decodeResponseBody(resp *http.Response) ([]byte, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	enc := string(resp.Header.Get("Content-Encoding"))
	switch enc {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	case "deflate":
		reader, err := zlib.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	case "br":
		reader := brotli.NewReader(bytes.NewReader(body))
		return io.ReadAll(reader)
	case "zstd":
		reader, err := zstd.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	default:
		return body, nil
	}
}

func (c *ClientHTTP2) getStatusClass(statusCode int) string {
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

func (c *ClientHTTP2) ProxyAddress() string {
	return c.proxyAddr
}

func (c *ClientHTTP2) IPAddress() string {
	return c.ipAddress
}
