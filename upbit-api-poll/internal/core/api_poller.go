package core

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/client/httpapi"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/fn"
)

var (
	ErrNoProxies = errors.New("no proxies were provided")
	ErrNoNews    = errors.New("no news found")
)

const (
	MetricUpbitNewsRequestsTotal    = "upbit_news_requests_total"
	MetricUpbitNewsErrorsTotal      = "upbit_news_errors_total"
	MetricUpbitNewsFetchedTotal     = "upbit_news_fetched_total"
	MetricUpbitNewNewsDetectedTotal = "upbit_new_news_detected_total"
	MetricUpbitNewsRequestDuration  = "upbit_news_request_duration"
	MetricUpbitNewsParseDuration    = "upbit_news_parse_duration"
	MetricUpbitNewsPollDuration     = "upbit_news_poll_duration"
)

type APIPoller struct {
	targetRPS float64

	proxies      []*httpapi.UpbitClient
	workSchedule entity.WorkSchedule

	latestNews      entity.News
	latestNewsGuard sync.Mutex

	deps di.Container
}

func NewAPIPoller(deps di.Container) (*APIPoller, error) {
	if len(deps.Config.APIPoller.Proxies) == 0 {
		return nil, ErrNoProxies
	}

	proxies := fn.Map(deps.Config.APIPoller.Proxies, func(proxy entity.Proxy) *httpapi.UpbitClient {
		client, err := httpapi.NewUpbitClient(
			deps.Config.UpbitClient,
			&proxy,
			deps.Logger,
			deps.Metrics,
		)
		if err != nil {
			panic(err)
		}

		return client
	})

	minExpectedProxiesCount := max(1, int(math.Ceil(deps.Config.APIPoller.TargetRPS/deps.Config.APIPoller.SingleProxyMaxRPS)))
	if len(proxies) < minExpectedProxiesCount {
		return nil, fmt.Errorf("expected at least %d proxies, got %d", minExpectedProxiesCount, len(proxies))
	}

	poller := &APIPoller{
		proxies:      proxies,
		workSchedule: deps.Config.APIPoller.WorkSchedule,
		targetRPS:    deps.Config.APIPoller.TargetRPS,
		deps:         deps,
	}

	hostIPClient, err := httpapi.NewUpbitClient(
		deps.Config.UpbitClient,
		nil,
		deps.Logger,
		deps.Metrics,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create host IP client: %w", err)
	}

	latestNews, err := poller.pollNews(context.Background(), hostIPClient)
	if err != nil {
		return nil, fmt.Errorf("failed to poll latest news: %w", err)
	}

	poller.latestNews = latestNews

	return poller, nil
}

func (p *APIPoller) StreamNews(ctx context.Context) <-chan entity.NewsTitle {
	newsChan := make(chan entity.NewsTitle, len(p.proxies)<<1)

	go func() {
		for {
			if !func() bool {
				defer func() {
					if r := recover(); r != nil {
						p.deps.Logger.Error("panic in populateNewsChannel", "error", r)
						p.deps.SendMessage("Panic in populateNewsChannel: %v", r)
					}
				}()

				select {
				case <-ctx.Done():
					return false
				default:
					p.populateNewsChannel(ctx, newsChan)
					return true
				}
			}() {
				break
			}
		}
	}()

	return newsChan
}

func (p *APIPoller) populateNewsChannel(ctx context.Context, newsChan chan<- entity.NewsTitle) {
	pollingInterval := time.Duration(float64(time.Second) / p.targetRPS)
	p.deps.Logger.Info("Polling news with interval", "interval", pollingInterval.String())

	p.deps.SendMessage(
		"Service is running.\nTarget RPS: %f\nSingle proxy max RPS: %f\nProxies count: %d\nPolling interval: %s",
		p.targetRPS,
		p.deps.Config.APIPoller.SingleProxyMaxRPS,
		len(p.proxies),
		pollingInterval.String(),
	)

	wg := sync.WaitGroup{}

	for i := 0; ; i = (i + 1) % len(p.proxies) {
		select {
		case <-ctx.Done():
			p.deps.Logger.Info(
				"Context cancelled, waiting for goroutines to finish",
			)
			wg.Wait()
			close(newsChan)
			return

		default:
			wg.Add(1)
			go func(proxy *httpapi.UpbitClient) {
				defer wg.Done()

				p.pollNewsAndUpdateIfDifferent(ctx, proxy, newsChan)
			}(p.proxies[i])

			time.Sleep(pollingInterval)
		}
	}
}

func (p *APIPoller) pollNewsAndUpdateIfDifferent(
	ctx context.Context,
	proxy *httpapi.UpbitClient,
	newsChan chan<- entity.NewsTitle,
) {
	p.deps.Metrics.IncrementCounter(MetricUpbitNewsRequestsTotal)

	currentNews, err := p.pollNews(ctx, proxy)
	if err != nil {
		p.deps.Metrics.IncrementCounter(MetricUpbitNewsErrorsTotal)
		p.deps.Logger.Error(
			"Failed to poll news",
			"error",
			err,
		)
		return
	}

	p.deps.Metrics.IncrementCounter(MetricUpbitNewsFetchedTotal)

	p.updateNewsIfDifferent(currentNews, newsChan)
}

func (p *APIPoller) updateNewsIfDifferent(
	currentNews entity.News,
	newsChan chan<- entity.NewsTitle,
) {
	p.latestNewsGuard.Lock()
	defer p.latestNewsGuard.Unlock()

	if currentNews.Title != p.latestNews.Title {
		now := time.Now()

		newsChan <- currentNews.Title

		listedAtDelay := now.Sub(currentNews.ListedAt)
		firstListedAtDelay := now.Sub(currentNews.FirstListedAt)
		receivedFromAPIDelay := now.Sub(currentNews.ReceivedFromAPIAt)
		timeGoneSinceLastNews := now.Sub(p.latestNews.ReceivedFromAPIAt)

		p.latestNews = currentNews

		p.deps.Metrics.IncrementCounter(MetricUpbitNewNewsDetectedTotal)

		p.deps.Logger.Info(
			"News detected",
			"title", currentNews.Title,
			"listed_at", currentNews.ListedAt.Format(time.RFC3339),
			"first_listed_at", currentNews.FirstListedAt.Format(time.RFC3339),
			"received_from_api_at", currentNews.ReceivedFromAPIAt.Format(time.RFC3339),
		)

		p.deps.SendMessage(
			"News detected: %s\n"+
				"Listed at: %s\n"+
				"First listed at: %s\n"+
				"Received from API at: %s\n"+
				"Listed at delay: %s\n"+
				"First listed at delay: %s\n"+
				"Received from API delay: %s\n"+
				"Time gone since last news: %s",
			currentNews.Title,
			currentNews.ListedAt.Format(time.RFC3339),
			currentNews.FirstListedAt.Format(time.RFC3339),
			currentNews.ReceivedFromAPIAt.Format(time.RFC3339),
			listedAtDelay.String(),
			firstListedAtDelay.String(),
			receivedFromAPIDelay.String(),
			timeGoneSinceLastNews.String(),
		)
	}
}

func (p *APIPoller) pollNews(
	ctx context.Context,
	proxy *httpapi.UpbitClient,
) (entity.News, error) {
	timer := p.deps.Metrics.StartTimer(MetricUpbitNewsPollDuration)
	defer timer.ObserveDuration()

	response, err := p.fetchNews(ctx, proxy)
	if err != nil {
		return entity.News{}, fmt.Errorf("failed to poll news: %w", err)
	}

	currentNews, err := p.parseNews(response)
	if err != nil {
		return entity.News{}, fmt.Errorf("failed to parse news: %w", err)
	}

	return currentNews, nil
}

func (p *APIPoller) fetchNews(
	ctx context.Context,
	proxy *httpapi.UpbitClient,
) (entity.UpbitResponse, error) {
	timer := p.deps.Metrics.StartTimer(MetricUpbitNewsRequestDuration)
	defer timer.ObserveDuration()

	response, err := proxy.Request(ctx)
	if err != nil {
		return entity.UpbitResponse{}, fmt.Errorf("failed to request news: %w", err)
	}

	return response, nil
}

func (p *APIPoller) parseNews(response entity.UpbitResponse) (entity.News, error) {
	timer := p.deps.Metrics.StartTimer(MetricUpbitNewsParseDuration)
	defer timer.ObserveDuration()

	var announcements entity.Announcements
	if err := announcements.UnmarshalJSON(response.Body); err != nil {
		return entity.News{}, fmt.Errorf(
			"failed to unmarshal announcements: %w, body: %s",
			err,
			string(response.Body),
		)
	}

	if len(announcements.Data.Notices) == 0 {
		return entity.News{}, ErrNoNews
	}

	return entity.News{
		ID:                response.ID,
		Title:             announcements.Data.Notices[0].Title,
		ListedAt:          announcements.Data.Notices[0].ListedAt,
		FirstListedAt:     announcements.Data.Notices[0].FirstListedAt,
		ReceivedFromAPIAt: response.CreatedAt,
	}, nil
}
