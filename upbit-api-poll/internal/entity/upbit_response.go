package entity

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type UpbitResponse struct {
	ID        uuid.UUID   `json:"id"`
	CreatedAt time.Time   `json:"created_at"`
	Status    int         `json:"status"`
	Headers   http.Header `json:"headers"`
	Body      []byte      `json:"body"`
	ProxyAddr string      `json:"proxy_addr"`
}

func NewUpbitResponse(
	status int,
	headers http.Header,
	body []byte,
	proxyAddr string,
) UpbitResponse {
	return UpbitResponse{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		Status:    status,
		Headers:   headers,
		Body:      body,
		ProxyAddr: proxyAddr,
	}
}

func (r *UpbitResponse) IsOK() bool {
	return r.Status == http.StatusOK
}

func (r *UpbitResponse) IsTooManyRequests() bool {
	return r.Status == http.StatusTooManyRequests
}

func (r *UpbitResponse) RequestAfter() time.Duration {
	retryAfter := r.Headers.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	retryAfterInt, err := strconv.Atoi(retryAfter)
	if err != nil {
		return 0
	}

	return time.Duration(retryAfterInt) * time.Second
}
