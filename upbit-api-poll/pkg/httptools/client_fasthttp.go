package httptools

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/valyala/fasthttp"
)

type ClientFastHTTP struct {
	ipAddress  string
	proxyAddr  string
	httpClient *fasthttp.Client

	maxRetries           int
	retryDelay           time.Duration
	retryDelayMultiplier float64

	metrics Metrics

	logger *slog.Logger
}

// FasthttpHTTPDialer creates a dial function for HTTP proxies that properly handles HTTPS
func FasthttpHTTPDialer(proxyAddr string) fasthttp.DialFunc {
	return func(addr string) (net.Conn, error) {
		// Parse proxy URL to handle authentication if present
		var auth string
		var proxyHost string

		// Parse the proxy URL properly
		if strings.Contains(proxyAddr, "://") {
			// Handle full URL format like http://user:pass@proxy:port
			u, err := url.Parse(proxyAddr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse proxy URL %s: %w", proxyAddr, err)
			}

			proxyHost = u.Host
			if u.User != nil {
				if password, ok := u.User.Password(); ok {
					auth = fmt.Sprintf("%s:%s", u.User.Username(), password)
				} else {
					auth = u.User.Username()
				}
			}
		} else if strings.Contains(proxyAddr, "@") {
			// Handle user:pass@proxy:port format
			parts := strings.Split(proxyAddr, "@")
			if len(parts) == 2 {
				auth = parts[0]
				proxyHost = parts[1]
			} else {
				proxyHost = proxyAddr
			}
		} else {
			proxyHost = proxyAddr
		}

		// Connect to proxy
		conn, err := fasthttp.Dial(proxyHost)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to proxy %s: %w", proxyHost, err)
		}

		// Send CONNECT request for HTTPS tunneling
		req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\n", addr)
		req += fmt.Sprintf("Host: %s\r\n", addr)
		req += "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36\r\n"
		req += "Proxy-Connection: keep-alive\r\n"

		if auth != "" {
			// Basic auth encoding using proper base64
			encoded := base64.StdEncoding.EncodeToString([]byte(auth))
			req += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", encoded)
		}
		req += "\r\n"

		if _, err := conn.Write([]byte(req)); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to write CONNECT request: %w", err)
		}

		// Read response line by line to handle proxy response properly
		reader := bufio.NewReader(conn)

		// Read status line
		statusLine, _, err := reader.ReadLine()
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to read CONNECT response status: %w", err)
		}

		// Parse status code from response
		statusParts := strings.Fields(string(statusLine))
		if len(statusParts) < 2 {
			conn.Close()
			return nil, fmt.Errorf("invalid CONNECT response format: %s", string(statusLine))
		}

		statusCode, err := strconv.Atoi(statusParts[1])
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("invalid status code in CONNECT response: %s", statusParts[1])
		}

		// Read headers until empty line
		for {
			line, _, err := reader.ReadLine()
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("failed to read CONNECT response headers: %w", err)
			}
			if len(line) == 0 {
				break // End of headers
			}
		}

		// Handle different status codes
		switch statusCode {
		case 200:
			// Success - connection established
			return conn, nil
		case 407:
			conn.Close()
			if auth == "" {
				return nil, fmt.Errorf("proxy authentication required but no credentials provided")
			}
			return nil, fmt.Errorf("proxy authentication failed - check username/password")
		case 403:
			conn.Close()
			return nil, fmt.Errorf("proxy connection forbidden - access denied")
		case 404:
			conn.Close()
			return nil, fmt.Errorf("proxy server not found")
		case 502:
			conn.Close()
			return nil, fmt.Errorf("proxy bad gateway - cannot reach target server")
		default:
			conn.Close()
			return nil, fmt.Errorf("proxy CONNECT failed with status %d", statusCode)
		}
	}
}

func NewClientFastHTTP(
	config ...ClientConfig,
) (*ClientFastHTTP, error) {
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

	client := &fasthttp.Client{
		MaxConnsPerHost:               100,
		ReadTimeout:                   ResponseTimeout,
		WriteTimeout:                  ResponseTimeout,
		DisablePathNormalizing:        true,
		DisableHeaderNamesNormalizing: true,
		NoDefaultUserAgentHeader:      true,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	proxyAddr := "direct"
	if cfg.Proxy != nil && cfg.Proxy.String() != "" {
		proxyAddr = cfg.Proxy.String()
		// Use the proper HTTP proxy dialer
		client.Dial = FasthttpHTTPDialer(cfg.Proxy.String())
	}

	return &ClientFastHTTP{
		logger:               cfg.Logger,
		metrics:              cfg.Metrics,
		maxRetries:           cfg.MaxRetries,
		retryDelay:           cfg.RetryDelay,
		retryDelayMultiplier: cfg.RetryDelayMultiplier,
		proxyAddr:            proxyAddr,
		ipAddress:            cfg.Proxy.Host,
		httpClient:           client,
	}, nil
}

func (u *ClientFastHTTP) Request(
	ctx context.Context,
	url string,
) (Response, error) {
	requestedAt := time.Now()

	timer := u.metrics.StartTimer(MetricUpbitRequestDuration, "proxy", u.proxyAddr)
	defer timer.ObserveDuration()

	u.metrics.IncrementCounter(MetricUpbitRequestsTotal, "proxy", u.proxyAddr)

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(url)
	req.Header.SetMethod("GET")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept-Language", "ko-KR, ko;q=1, en-US;q=0.1")
	req.Header.Set(
		"Cache-Control",
		"no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0",
	)
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")
	req.Header.Set("Origin", fmt.Sprintf("https://%s.com", uuid.New().String()))

	var lastErr error
	delay := u.retryDelay

	for i := 0; i <= u.maxRetries; i++ {
		err := u.httpClient.Do(req, resp)
		if err == nil {
			break
		}

		lastErr = err

		// Log specific proxy authentication errors
		if strings.Contains(err.Error(), "407") ||
			strings.Contains(err.Error(), "Proxy Authentication Required") {
			u.logger.Error(
				"Proxy authentication failed",
				"attempt",
				i+1,
				"max_retries",
				u.maxRetries,
				"error",
				err.Error(),
				"proxy",
				u.proxyAddr,
				"suggestion",
				"Check proxy credentials in proxy URL format: http://username:password@proxy:port",
			)
		} else {
			u.logger.Debug("Request failed, retrying",
				"attempt", i+1,
				"max_retries", u.maxRetries,
				"error", err.Error(),
				"proxy", u.proxyAddr)
		}

		if i < u.maxRetries {
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * u.retryDelayMultiplier)
		}
	}

	if lastErr != nil {
		u.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			u.proxyAddr,
			"type",
			"request_failed",
		)
		return Response{}, fmt.Errorf(
			"request failed after %d retries: %w",
			u.maxRetries,
			lastErr,
		)
	}

	body, err := u.decodeResponseBody(resp)
	if err != nil {
		u.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			u.proxyAddr,
			"type",
			"body_read",
		)
		return Response{}, fmt.Errorf("failed to decode response body: %w", err)
	}

	u.metrics.IncrementCounter(MetricUpbitResponsesTotal,
		"proxy", u.proxyAddr,
		"status_code", strconv.Itoa(resp.StatusCode()),
		"status_class", u.getStatusClass(resp.StatusCode()))

	upbitResp := NewHTTPResponse(
		requestedAt,
		resp.StatusCode(),
		u.convertHeaders(&resp.Header),
		body,
		u.proxyAddr,
		"fasthttp",
	)

	// Handle rate limiting
	if resp.StatusCode() == http.StatusTooManyRequests {
		u.metrics.IncrementCounter(MetricUpbitRateLimitedTotal, "proxy", u.proxyAddr)
		u.metrics.IncrementCounter(
			MetricUpbitErrorsTotal,
			"proxy",
			u.proxyAddr,
			"type",
			"rate_limited",
		)

		u.logger.Info(
			"Rate limited response",
			"Cf-Cache-Status", upbitResp.Headers.Get("Cf-Cache-Status"),
			"Cf-Ray", upbitResp.Headers.Get("Cf-Ray"),
			"X-Request-ID", upbitResp.Headers.Get("X-Request-ID"),
			"X-Runtime", upbitResp.Headers.Get("X-Runtime"),
			"Retry-After", upbitResp.Headers.Get("Retry-After"),
			"proxy", u.proxyAddr,
		)

		return upbitResp, nil
	}

	// Handle successful response
	if resp.StatusCode() == http.StatusOK {
		u.metrics.IncrementCounter(MetricUpbitSuccessfulRequests, "proxy", u.proxyAddr)
		return upbitResp, nil
	}

	return upbitResp, nil
}

func (u *ClientFastHTTP) decodeResponseBody(resp *fasthttp.Response) ([]byte, error) {
	enc := string(resp.Header.Peek("Content-Encoding"))
	switch enc {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(resp.Body()))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	case "deflate":
		reader, err := zlib.NewReader(bytes.NewReader(resp.Body()))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	case "br":
		reader := brotli.NewReader(bytes.NewReader(resp.Body()))
		return io.ReadAll(reader)
	case "zstd":
		reader, err := zstd.NewReader(bytes.NewReader(resp.Body()))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(reader)
	default:
		return resp.Body(), nil // no encoding
	}
}

func (u *ClientFastHTTP) convertHeaders(header *fasthttp.ResponseHeader) http.Header {
	h := make(http.Header)
	header.VisitAll(func(key, value []byte) {
		h.Set(string(key), string(value))
	})
	return h
}

func (u *ClientFastHTTP) getStatusClass(statusCode int) string {
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

func (u *ClientFastHTTP) ProxyAddress() string {
	return u.proxyAddr
}

func (u *ClientFastHTTP) IPAddress() string {
	return u.ipAddress
}
