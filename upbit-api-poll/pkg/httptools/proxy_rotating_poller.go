package httptools

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrAlreadyPolling = errors.New("already polling")
	ErrNoProxies      = errors.New("no proxies were provided")
)

const (
	MetricUpbitNewsRequestsTotal   = "upbit_news_requests_total"
	MetricUpbitNewsErrorsTotal     = "upbit_news_errors_total"
	MetricUpbitNewsFetchedTotal    = "upbit_news_fetched_total"
	MetricUpbitNewsRequestDuration = "upbit_news_request_duration"
	MetricUpbitNewsPollDuration    = "upbit_news_poll_duration"
)

// ProxyRotatingPoller is a poller that rotates through a list of proxies to poll news.
// To initialize it, use ProxyRotatingPollerBuilder.
type ProxyRotatingPoller struct {
	urlGuard sync.RWMutex
	url      string

	targetRPS         float64
	singleProxyMaxRPS float64

	clientsByLocation ClientsPool
	totalClientsCount int

	notifier Notifier

	workSchedule WorkSchedule

	running atomic.Bool

	logger  *slog.Logger
	metrics Metrics
}

type Client interface {
	Request(ctx context.Context, url string) (Response, error)
	ProxyAddress() string
	IPAddress() string
}

type ClientsPool interface {
	Acquire() ([]Client, ReleaseFunc)
	Len() int
}

type ReleaseFunc func()

type Notifier interface {
	SendMessage(message string, args ...interface{})
}

type Metrics interface {
	// Counters
	IncrementCounter(name string, labels ...string)

	// Gauges
	SetGauge(name string, value float64, labels ...string)
	IncrementGauge(name string, labels ...string)
	DecrementGauge(name string, labels ...string)

	// Histograms
	ObserveHistogram(name string, value float64, labels ...string)
	StartTimer(name string, labels ...string) Timer
}

type WorkSchedule interface {
	WorkNow() bool
	NextWorkSession() (time.Duration, error)
}

type Timer interface {
	ObserveDuration()
}

func (p *ProxyRotatingPoller) SetURL(url string) {
	p.urlGuard.Lock()
	defer p.urlGuard.Unlock()

	p.url = url

	// p.notifier.SendMessage("Polling URL changed to %s", url)
}

func (p *ProxyRotatingPoller) StartPolling(ctx context.Context) (<-chan Response, error) {
	if !p.running.CompareAndSwap(false, true) {
		return nil, ErrAlreadyPolling
	}

	responsesChan := make(chan Response, p.totalClientsCount<<1)

	go func() {
		defer close(responsesChan)

		p.doPollingWithRecovery(ctx, responsesChan)
	}()

	return responsesChan, nil
}

func (p *ProxyRotatingPoller) doPollingWithRecovery(
	ctx context.Context,
	responsesChan chan<- Response,
) {
	for {
		shouldContinue := p.executePollingWithRecovery(ctx, responsesChan)
		if !shouldContinue {
			return
		}
	}
}

func (p *ProxyRotatingPoller) executePollingWithRecovery(
	ctx context.Context,
	responsesChan chan<- Response,
) (shouldContinue bool) {
	defer func() {
		if r := recover(); r != nil {
			p.logger.Error("panic in populateWithResponses", "error", r)
			shouldContinue = true // Continue execution after panic recovery
		}
	}()

	err := p.populateWithResponses(ctx, responsesChan)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		p.logger.Error("Failed to populate with responses", "error", err)
		return false // Stop execution on context cancellation
	}

	return true // Continue execution
}

func (p *ProxyRotatingPoller) populateWithResponses(
	ctx context.Context,
	responsesChan chan<- Response,
) error {
	pollingInterval := time.Duration(float64(time.Second) / p.targetRPS)
	p.logger.Info("Polling news with interval", "interval", pollingInterval.String())

	p.notifier.SendMessage(
		"Service is running.\nTarget RPS: %f\nSingle proxy max RPS: %f\nProxies count: %d\nPolling interval: %s\nURL: %s",
		p.targetRPS,
		p.singleProxyMaxRPS,
		p.totalClientsCount,
		pollingInterval.String(),
		p.url,
	)

	wg := sync.WaitGroup{}
	workDebt := time.Duration(0)

	nextWorkSession := func() time.Duration {
		sleepTillNextWorkTime, err := p.workSchedule.NextWorkSession()
		if err != nil {
			p.logger.Error("Failed to get next work session", "error", err)
			return 0
		}

		return sleepTillNextWorkTime
	}()

	lastPollStartTime := time.Now().Add(nextWorkSession)

	failedToFetchResponseCount := atomic.Int64{}

	for {
		if !p.workSchedule.WorkNow() {
			sleepTillNextWorkTime, err := p.workSchedule.NextWorkSession()
			if err != nil {
				p.logger.Error("Failed to get next work session", "error", err)
				return nil
			}

			p.notifier.SendMessage(
				"Service is not working now. Next work session in %s",
				sleepTillNextWorkTime.String(),
			)

			time.Sleep(sleepTillNextWorkTime)

			p.notifier.SendMessage(
				"Poller was resumed after sleep",
			)
		}

		select {
		case <-ctx.Done():
			p.logger.Info(
				"Context cancelled, waiting for goroutines to finish",
			)
			wg.Wait()

			return nil

		default:
			clients, releaseClients := p.clientsByLocation.Acquire()

			wg.Add(1)
			go func(clients []Client) {
				timer := p.metrics.StartTimer(MetricUpbitNewsPollDuration)
				defer timer.ObserveDuration()

				defer wg.Done()
				defer releaseClients()

				localWg := sync.WaitGroup{}

				for _, client := range clients {
					localWg.Add(1)
					go func(client Client) {
						defer localWg.Done()
						response, err := p.fetchResponse(ctx, client)
						if err != nil {
							failedToFetchResponseCount.Add(1)
							if failedToFetchResponseCount.Load() > 249 {
								p.logger.Error("Failed to fetch response", "error", err, "countOfSuppressedErrors", failedToFetchResponseCount.Load())
								failedToFetchResponseCount.Store(0)
								return
							}
						}

						responsesChan <- response
					}(client)
				}

				localWg.Wait()
			}(clients)

			// Calculate time to sleep before next poll.
			// Elapsed time is the time since the last poll start time.
			// Work debt is the time that the poller is behind the target RPS.
			// Polling interval is the theoretical time between polls. (Time between polls if there was no work debt)
			// Time to sleep before next poll is the time to sleep before the next poll.
			// If the time to sleep before next poll is negative, it means that the poller is ahead of the target RPS.
			// In this case, the poller will sleep for 0 seconds.
			// If the time to sleep before next poll is positive, it means that the poller is behind the target RPS.
			elapsed := time.Since(lastPollStartTime)
			timeToSleepBeforeNextPoll := pollingInterval - elapsed - workDebt
			workDebt = 0
			if timeToSleepBeforeNextPoll < 0 {
				workDebt = -timeToSleepBeforeNextPoll
				timeToSleepBeforeNextPoll = 0
			}

			if timeToSleepBeforeNextPoll > 0 {
				time.Sleep(timeToSleepBeforeNextPoll)
			}

			lastPollStartTime = time.Now()
		}
	}
}

func (p *ProxyRotatingPoller) fetchResponse(
	ctx context.Context,
	client Client,
) (Response, error) {
	timer := p.metrics.StartTimer(MetricUpbitNewsRequestDuration)
	defer timer.ObserveDuration()

	p.metrics.IncrementCounter(MetricUpbitNewsRequestsTotal)

	p.urlGuard.RLock()
	defer p.urlGuard.RUnlock()

	response, err := client.Request(ctx, p.url)
	if err != nil {
		p.metrics.IncrementCounter(MetricUpbitNewsErrorsTotal)
		return Response{}, fmt.Errorf("failed to request news: %w", err)
	}

	p.metrics.IncrementCounter(MetricUpbitNewsFetchedTotal)

	return response, nil
}
