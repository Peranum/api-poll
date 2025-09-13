package httptools

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type Response struct {
	ID          uuid.UUID   `json:"id"`
	RequestedAt time.Time   `json:"requested_at"`
	ReceivedAt  time.Time   `json:"received_at"`
	StatusCode  int         `json:"status_code"`
	Headers     http.Header `json:"headers"`
	Body        []byte      `json:"body"`
	ProxyAddr   string      `json:"proxy_addr"`
	ClientName  string      `json:"client_name"`
}

func NewHTTPResponse(
	requestedAt time.Time,
	status int,
	headers http.Header,
	body []byte,
	proxyAddr string,
	clientName string,
) Response {
	return Response{
		ID:          uuid.New(),
		RequestedAt: requestedAt,
		ReceivedAt:  time.Now(),
		StatusCode:  status,
		Headers:     headers,
		Body:        body,
		ProxyAddr:   proxyAddr,
		ClientName:  clientName,
	}
}

func (r *Response) IsOK() bool {
	return r.StatusCode == http.StatusOK
}

func (r *Response) IsTooManyRequests() bool {
	return r.StatusCode == http.StatusTooManyRequests
}

func (r *Response) RequestAfter() time.Duration {
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
