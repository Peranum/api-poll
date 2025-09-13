package core

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/httptools"
)

const (
	noticeFetcherChanSize = 8

	defaultNoticeTitle = "비트코인, 이더리움, 엑스알피(리플), NFT 등 다양한 디지털 자산, 국내 거래량 1위 거래소 업비트에서 지금 확인해보세요. No.1 Digital Asset Exchange in Korea, Upbit. Trade various digital assets conveniently and securely including Bitcoin, Ethereum, XRP(Ripple), NFT etc."

	MetricUpbitNewNewsDetectedTotal = "upbit_new_news_detected_total"
	MetricUpbitNewsParseDuration    = "upbit_news_parse_duration"
)

type NoticeByIDFetcher struct {
	poller *httptools.ProxyRotatingPoller

	newsGuard     sync.Mutex
	lastNewsTitle string
	nextNewsID    int

	alreadyStreaming atomic.Bool

	deps di.Container
}

func NewNoticeByIDFetcher(deps di.Container) (*NoticeByIDFetcher, error) {
	fetcher := &NoticeByIDFetcher{deps: deps}

	if err := fetcher.initLatestNewsTitleAndNextNewsID(context.Background()); err != nil {
		fetcher.deps.Logger.Error("failed to init latest news title and next news id", "error", err)
		return nil, err
	}

	if err := fetcher.initPoller(); err != nil {
		fetcher.deps.Logger.Error("failed to init poller", "error", err)
		return nil, err
	}

	return fetcher, nil
}

func (f *NoticeByIDFetcher) StreamNewNoticeTitles(
	ctx context.Context,
) (<-chan entity.NewsTitle, error) {
	if !f.alreadyStreaming.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("already streaming")
	}

	f.deps.Logger.Info("Starting to stream new notice titles", "next_news_id", f.nextNewsID)

	newsChan := make(chan entity.NewsTitle, noticeFetcherChanSize)

	responsesChan, err := f.poller.StartPolling(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(newsChan)

		if err := f.populateWithNewNews(ctx, newsChan, responsesChan); err != nil {
			f.deps.Logger.Error("failed to populate news chan", "error", err)
		}
	}()

	return newsChan, nil
}

func (f *NoticeByIDFetcher) populateWithNewNews(
	ctx context.Context,
	newsChan chan<- entity.NewsTitle,
	responses <-chan httptools.Response,
) error {
	banned := map[string]time.Time{}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case response := <-responses:
			if response.IsTooManyRequests() {
				if _, ok := banned[response.ProxyAddr]; ok {
					if time.Since(banned[response.ProxyAddr]) < 1*time.Hour {
						continue
					}

					delete(banned, response.ProxyAddr)
					continue
				}

				banned[response.ProxyAddr] = time.Now()

				f.deps.SendMessage(
					"Something is wrong with poller: too many requests\nProxy: %s\nStatus: %d\nHeaders: %s\nDead proxies count: %d",
					response.ProxyAddr,
					response.StatusCode,
					response.Headers,
					len(banned),
				)

				f.deps.Logger.Error(
					"too many requests",
					"status",
					response.StatusCode,
					"proxy",
					response.ProxyAddr,
					"headers",
					response.Headers,
				)

				continue
			}

			if !response.IsOK() {
				continue
			}

			title, err := f.parseNoticePage(response)
			if err != nil {
				f.deps.Logger.Error(
					"failed to parse notice page",
					"error",
					err,
					"body",
					string(response.Body),
				)
				continue
			}

			f.updateAndNotify(newsChan, title, response)
		}
	}
}

func (f *NoticeByIDFetcher) parseNoticePage(response httptools.Response) (entity.NewsTitle, error) {
	timer := f.deps.Metrics.StartTimer(MetricUpbitNewsParseDuration, "fetcher", "notice_by_id")
	defer timer.ObserveDuration()

	// Convert the response body to a string
	bodyStr := string(response.Body)

	// Find the start of the meta tag with name="description"
	metaTagStart := `<meta name="description" content="`
	startIndex := strings.Index(bodyStr, metaTagStart)
	if startIndex == -1 {
		return "", fmt.Errorf("description meta tag not found")
	}

	// Find the end of the content attribute
	startIndex += len(metaTagStart)
	endIndex := strings.Index(bodyStr[startIndex:], `" />`)
	if endIndex == -1 {
		return "", fmt.Errorf("end of description content not found")
	}

	// Extract the description content
	description := bodyStr[startIndex : startIndex+endIndex]

	return entity.NewsTitle(description), nil
}

func (f *NoticeByIDFetcher) updateAndNotify(
	newsChan chan<- entity.NewsTitle,
	title entity.NewsTitle,
	response httptools.Response,
) {
	if title == defaultNoticeTitle {
		return
	}

	f.newsGuard.Lock()
	defer f.newsGuard.Unlock()

	if f.lastNewsTitle == title {
		f.deps.Logger.Info(
			"skipping notice",
			"title",
			title,
		)
		return
	}

	f.nextNewsID += 1
	f.lastNewsTitle = title

	newsChan <- title
	f.poller.SetURL(fmt.Sprintf(f.deps.Config.UpbitAPI.NoticeByIDEndpoint, f.nextNewsID))

	f.deps.Metrics.IncrementCounter(MetricUpbitNewNewsDetectedTotal, "fetcher", "notice_by_id")

	f.deps.SendMessage(
		fmt.Sprintf(
			"New notice\n"+
				"* NEWS INFO\n"+
				"Title: %s\n"+
				"\n"+
				"* RESPONSE INFO\n"+
				"Client: %s\n"+
				"Proxy address: %s\n"+
				"\n"+
				"Requested at: %s\n"+
				"Received at: %s\n"+
				"\n"+
				"Status: %d\n"+
				"Headers: %s\n"+
				"\n"+
				"* DELAYS INFO\n"+
				"Between requested_at and received_at: %s\n",
			title,
			response.ClientName,
			response.ProxyAddr,
			response.RequestedAt.Format("2006-01-02 15:04:05.000"),
			response.ReceivedAt.Format("2006-01-02 15:04:05.000"),
			response.StatusCode,
			response.Headers,
			response.ReceivedAt.Sub(response.RequestedAt),
		),
	)
}

func (f *NoticeByIDFetcher) initLatestNewsTitleAndNextNewsID(ctx context.Context) error {
	hostIPClient, err := httptools.NewClientHTTP2(httptools.ClientConfig{
		Logger:  f.deps.Logger,
		Metrics: &metricsAdapter{prometheus: f.deps.Metrics},
	})
	if err != nil {
		return err
	}

	response, err := hostIPClient.Request(ctx, fmt.Sprintf(f.deps.Config.UpbitAPI.AnnouncementsEndpoint, 20))
	if err != nil {
		f.deps.Logger.Error("failed to get notices", "error", err)
		return err
	}

	f.deps.Logger.Info("getting notices", "endpoint", fmt.Sprintf(f.deps.Config.UpbitAPI.AnnouncementsEndpoint, 20), "body", string(response.Body), "status", response.StatusCode, "proxy", response.ProxyAddr)

	notices := entity.Announcements{}
	if err := notices.UnmarshalJSON(response.Body); err != nil {
		f.deps.Logger.Error("failed to unmarshal notices", "error", err)
		return err
	}

	if !notices.Success {
		return fmt.Errorf("failed to get notices: %s", notices.ErrorMessage)
	}

	if len(notices.Data.Notices) == 0 {
		return fmt.Errorf("no notices found")
	}

	announcements := notices.Data.Notices

	f.deps.Logger.Info("Fetched initial announcements", "count", len(notices.Data.Notices))

	slices.SortFunc(announcements, func(a, b entity.Notice) int {
		return a.ID - b.ID
	})

	lastNoticeIndex := len(announcements) - 1
	f.lastNewsTitle = announcements[lastNoticeIndex].Title
	f.nextNewsID = announcements[lastNoticeIndex].ID + 1

	pollingInterval := time.Duration(
		float64(time.Second) / f.deps.Config.UpbitAPI.NoticeByIDSingleIPMaxRPS,
	)

	timer := time.NewTicker(pollingInterval)
	defer timer.Stop()

	f.deps.Logger.Info(
		"Polling interval on finding last notice",
		"interval",
		pollingInterval.String(),
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-timer.C:

			response, err := hostIPClient.Request(
				ctx,
				fmt.Sprintf(f.deps.Config.UpbitAPI.NoticeByIDEndpoint, f.nextNewsID),
			)
			if err != nil {
				return err
			}

			title, err := f.parseNoticePage(response)
			if err != nil {
				return err
			}

			f.deps.Logger.Info("Fetching notice", "id", f.nextNewsID, "title", title)

			if title == defaultNoticeTitle {
				f.deps.Logger.Info(
					"no more notices",
					"next_news_id",
					f.nextNewsID,
					"last_news_title",
					f.lastNewsTitle,
				)
				return nil
			}

			f.nextNewsID += 1
			f.lastNewsTitle = title
		}
	}
}

func (f *NoticeByIDFetcher) initPoller() error {
	poller, err := httptools.NewProxyRotatingPollerBuilder().
		WithURL(fmt.Sprintf(f.deps.Config.UpbitAPI.NoticeByIDEndpoint, f.nextNewsID)).
		WithTargetRPS(f.deps.Config.ProxyRotatingPoller.TargetRPS).
		WithSingleProxyMaxRPS(f.deps.Config.UpbitAPI.NoticeByIDSingleIPMaxRPS).
		WithProxies(f.deps.Config.ProxyRotatingPoller.Proxies...).
		WithWorkSchedule(&f.deps.Config.ProxyRotatingPoller.WorkSchedule).
		WithLogger(f.deps.Logger).
		WithNotifier(f.deps).
		WithMetrics(&metricsAdapter{prometheus: f.deps.Metrics}).
		WithInitProxyClientFn(func(proxy httptools.Proxy) (httptools.Client, error) {
			return httptools.NewClientHTTP2(httptools.ClientConfig{
				Proxy:               &proxy,
				ClientRetriesConfig: f.deps.Config.ProxyRotatingPoller.Retries,
				Metrics:             &metricsAdapter{prometheus: f.deps.Metrics},
				Logger:              f.deps.Logger,
			})
		}).
		Build()
	if err != nil {
		return err
	}

	f.poller = poller

	return nil
}
