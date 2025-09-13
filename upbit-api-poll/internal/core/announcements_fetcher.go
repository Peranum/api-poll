package core

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/httptools"
)

type AnnouncementsFetcher struct {
	poller *httptools.ProxyRotatingPoller

	noticeGuard sync.Mutex
	lastNotice  entity.Notice

	alreadyStreaming atomic.Bool

	deps di.Container
}

func NewAnnouncementsFetcher(deps di.Container) (*AnnouncementsFetcher, error) {
	fetcher := &AnnouncementsFetcher{deps: deps}

	if err := fetcher.initLatestNotice(context.Background()); err != nil {
		return nil, err
	}

	if err := fetcher.initPoller(); err != nil {
		return nil, err
	}

	return fetcher, nil
}

func (f *AnnouncementsFetcher) StreamNewNewsTitles(
	ctx context.Context,
) (<-chan entity.NewsTitle, error) {
	if !f.alreadyStreaming.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("already streaming")
	}

	f.deps.Logger.Info("Starting to stream new announcements")

	announcementsChan := make(chan entity.NewsTitle, announcementFetcherChanSize)

	responsesChan, err := f.poller.StartPolling(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(announcementsChan)

		if err := f.populateWithNewAnnouncements(ctx, announcementsChan, responsesChan); err != nil {
			f.deps.Logger.Error("failed to populate announcements chan", "error", err)
		}
	}()

	return announcementsChan, nil
}

func (f *AnnouncementsFetcher) populateWithNewAnnouncements(
	ctx context.Context,
	announcementsChan chan<- entity.NewsTitle,
	responses <-chan httptools.Response,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case response := <-responses:
			if response.StatusCode != http.StatusOK {
				if response.StatusCode == http.StatusTooManyRequests {
					f.deps.SendMessage("Something is wrong with poller: too many requests")
					f.deps.Logger.Error("too many requests", "status", response.StatusCode)
					return fmt.Errorf("too many requests")
				}

				continue
			}

			announcement, err := f.parseAnnouncement(response)
			if err != nil {
				f.deps.Logger.Error("failed to parse announcement", "error", err)
				continue
			}

			f.updateAndNotify(announcementsChan, announcement, response)
		}
	}
}

func (f *AnnouncementsFetcher) updateAndNotify(
	announcementsChan chan<- entity.NewsTitle,
	announcement entity.Notice,
	response httptools.Response,
) {
	f.noticeGuard.Lock()
	defer f.noticeGuard.Unlock()

	if announcement.Title == f.lastNotice.Title {
		return
	}

	f.lastNotice = announcement
	announcementsChan <- announcement.Title
	f.deps.Metrics.IncrementCounter(MetricUpbitNewNewsDetectedTotal, "fetcher", "announcements")

	f.deps.Logger.Info("New announcement", "notice", announcement)

	f.deps.SendMessage(
		fmt.Sprintf(
			"New announcement\n"+
				"\n"+
				"* NEWS INFO\n"+
				"ID: %d\n"+
				"Title: %s\n"+
				"Category: %s\n"+
				"\n"+
				"Listed at: %s\n"+
				"First listed at: %s\n"+
				"\n"+
				"Link: %s\n"+
				"\n"+
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
				"\n"+
				"* DELAYS INFO\n"+
				"Between received_at and listed_at: %s\n"+
				"\n"+
				"Between requested_at and received_at: %s\n",
			announcement.ID,
			announcement.Title,
			announcement.Category,
			announcement.ListedAt,
			announcement.FirstListedAt,
			fmt.Sprintf(f.deps.Config.UpbitAPI.NoticeByIDEndpoint, announcement.ID),
			response.ClientName,
			response.ProxyAddr,
			response.RequestedAt.Format("2006-01-02 15:04:05.000"),
			response.ReceivedAt.Format("2006-01-02 15:04:05.000"),
			response.StatusCode,
			response.Headers,
			response.ReceivedAt.Sub(announcement.ListedAt),
			response.ReceivedAt.Sub(response.RequestedAt),
		),
	)
}

func (f *AnnouncementsFetcher) parseAnnouncement(response httptools.Response) (entity.Notice, error) {
	timer := f.deps.Metrics.StartTimer(MetricUpbitNewsParseDuration, "fetcher", "announcements")
	defer timer.ObserveDuration()

	notices := entity.Announcements{}
	if err := notices.UnmarshalJSON(response.Body); err != nil {
		return entity.Notice{}, err
	}

	if !notices.Success {
		return entity.Notice{}, fmt.Errorf("failed to get notices: %s", notices.ErrorMessage)
	}

	if len(notices.Data.Notices) == 0 {
		return entity.Notice{}, fmt.Errorf("no notices found")
	}

	announcements := notices.Data.Notices

	lastAnnouncement := announcements[len(announcements)-1]

	return lastAnnouncement, nil
}

func (f *AnnouncementsFetcher) initLatestNotice(ctx context.Context) error {
	hostIPClient, err := httptools.NewClientHTTP2(httptools.ClientConfig{
		Logger:  f.deps.Logger,
		Metrics: &metricsAdapter{prometheus: f.deps.Metrics},
	})
	if err != nil {
		return err
	}

	response, err := hostIPClient.Request(
		ctx,
		fmt.Sprintf(f.deps.Config.UpbitAPI.AnnouncementsEndpoint, 1),
	)
	if err != nil {
		return err
	}

	lastAnnouncement, err := f.parseAnnouncement(response)
	if err != nil {
		return err
	}

	f.lastNotice = lastAnnouncement

	f.deps.Logger.Info("Fetched initial announcements", "notice", f.lastNotice)

	return nil
}

func (f *AnnouncementsFetcher) initPoller() error {
	poller, err := httptools.NewProxyRotatingPollerBuilder().
		WithURL(fmt.Sprintf(f.deps.Config.UpbitAPI.AnnouncementsEndpoint, 1)).
		WithTargetRPS(f.deps.Config.ProxyRotatingPoller.TargetRPS).
		WithSingleProxyMaxRPS(f.deps.Config.UpbitAPI.AnnouncementsSingleIPMaxRPS).
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
