package core

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/httptools"
)

const announcementFetcherChanSize = 8

type AnnouncementByIDFetcher struct {
	poller *httptools.ProxyRotatingPoller

	newsGuard     sync.Mutex
	lastNewsTitle string
	nextNewsID    int

	alreadyStreaming atomic.Bool

	deps di.Container
}

func NewAnnouncementByIDFetcher(deps di.Container) (*AnnouncementByIDFetcher, error) {
	fetcher := &AnnouncementByIDFetcher{deps: deps}

	if err := fetcher.initLatestNewsTitleAndNextNewsID(context.Background()); err != nil {
		return nil, err
	}

	if err := fetcher.initPoller(); err != nil {
		return nil, err
	}

	return fetcher, nil
}

func (f *AnnouncementByIDFetcher) StreamNewAnnouncementTitles(
	ctx context.Context,
) (<-chan entity.NewsTitle, error) {
	if !f.alreadyStreaming.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("already streaming")
	}

	newsChan := make(chan entity.NewsTitle, announcementFetcherChanSize)

	f.poller.SetURL(fmt.Sprintf(f.deps.Config.UpbitAPI.AnnouncementByIDEndpoint, f.nextNewsID))

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

func (f *AnnouncementByIDFetcher) populateWithNewNews(
	ctx context.Context,
	newsChan chan<- entity.NewsTitle,
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

			announcement, ok := func() (entity.SingleAnnouncement, bool) {
				timer := f.deps.Metrics.StartTimer(
					MetricUpbitNewsParseDuration,
					"fetcher",
					"announcement_by_id",
				)
				defer timer.ObserveDuration()

				announcement := entity.SingleAnnouncement{}
				if err := announcement.UnmarshalJSON(response.Body); err != nil {
					f.deps.Logger.Error(
						"failed to unmarshal announcement",
						"error",
						err,
						"body",
						string(response.Body),
					)
					return entity.SingleAnnouncement{}, false
				}

				return announcement, true
			}()

			if !ok {
				continue
			}

			if announcement.Success {
				f.updateAndNotify(newsChan, announcement, response)
			}
		}
	}
}

func (f *AnnouncementByIDFetcher) updateAndNotify(
	newsChan chan<- entity.NewsTitle,
	announcement entity.SingleAnnouncement,
	response httptools.Response,
) {
	if !announcement.Success {
		f.deps.Logger.Error(
			"updateAndNotify accepts only successful announcements",
			"error",
			announcement.ErrorMessage,
		)
		return
	}

	f.newsGuard.Lock()
	defer f.newsGuard.Unlock()

	if f.lastNewsTitle == announcement.Data.Title {
		f.deps.Logger.Info(
			"skipping announcement",
			"id",
			announcement.Data.ID,
			"title",
			announcement.Data.Title,
		)
		return
	}

	f.nextNewsID = announcement.Data.ID + 1
	f.lastNewsTitle = announcement.Data.Title

	newsChan <- announcement.Data.Title
	f.poller.SetURL(fmt.Sprintf(f.deps.Config.UpbitAPI.AnnouncementByIDEndpoint, f.nextNewsID))

	f.deps.Metrics.IncrementCounter(
		MetricUpbitNewNewsDetectedTotal,
		"fetcher",
		"announcement_by_id",
	)

	f.deps.SendMessage(
		fmt.Sprintf(
			"!!!New announcement !!!\n"+
				"-----&gt; NEWS INFO &lt;----\n"+
				"Title: %s\n"+
				"Category: %s\n"+
				"\n"+
				"Listed at: %s\n"+
				"First listed at: %s\n"+
				"---&gt; RESPONSE INFO &lt;---\n"+
				"Client: %s\n"+
				"Proxy address: %s\n"+
				"\n"+
				"Requested at: %s\n"+
				"Received at: %s\n"+
				"\n"+
				"Status: %d\n"+
				"Headers: %s\n"+
				"\n"+
				"----&gt; DELAYS INFO &lt;----\n"+
				"Between requested_at and received_at: %s\n"+
				"Between requested_at and listed_at: %s\n"+
				"Between received_at and listed_at: %s\n",
			announcement.Data.Title,
			announcement.Data.Category,
			announcement.Data.ListedAt.Format("2006-01-02 15:04:05.000"),
			announcement.Data.FirstListedAt.Format("2006-01-02 15:04:05.000"),
			response.ClientName,
			response.ProxyAddr,
			response.RequestedAt.Format("2006-01-02 15:04:05.000"),
			response.ReceivedAt.Format("2006-01-02 15:04:05.000"),
			response.StatusCode,
			response.Headers,
			response.RequestedAt.Sub(response.ReceivedAt),
			response.RequestedAt.Sub(announcement.Data.ListedAt),
			response.ReceivedAt.Sub(announcement.Data.ListedAt),
		),
	)
}

func (f *AnnouncementByIDFetcher) initLatestNewsTitleAndNextNewsID(ctx context.Context) error {
	hostIPClient, err := httptools.NewClientHTTP2(httptools.ClientConfig{
		Logger:  f.deps.Logger,
		Metrics: &metricsAdapter{prometheus: f.deps.Metrics},
	})
	if err != nil {
		return err
	}

	response, err := hostIPClient.Request(ctx, f.deps.Config.UpbitAPI.AnnouncementsEndpoint)
	if err != nil {
		return err
	}

	announcements := entity.Announcements{}
	if err := announcements.UnmarshalJSON(response.Body); err != nil {
		return err
	}

	if !announcements.Success {
		return fmt.Errorf("failed to get announcements: %s", announcements.ErrorMessage)
	}

	if len(announcements.Data.Notices) == 0 {
		return fmt.Errorf("no announcements found")
	}

	notices := announcements.Data.Notices

	slices.SortFunc(notices, func(a, b entity.Notice) int {
		return a.ID - b.ID
	})

	lastNoticeIndex := len(notices) - 1
	f.lastNewsTitle = notices[lastNoticeIndex].Title
	f.nextNewsID = notices[lastNoticeIndex].ID + 1

	timer := time.NewTicker(
		time.Duration(float64(time.Second) / f.deps.Config.UpbitAPI.AnnouncementByIDSingleIPMaxRPS),
	)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-timer.C:
			response, err := hostIPClient.Request(
				ctx,
				fmt.Sprintf(f.deps.Config.UpbitAPI.AnnouncementByIDEndpoint, f.nextNewsID),
			)
			if err != nil {
				return err
			}

			announcement := entity.SingleAnnouncement{}
			if err := announcement.UnmarshalJSON(response.Body); err != nil {
				return err
			}

			if !announcement.Success {
				if announcement.ErrorCode == -1 {
					f.deps.Logger.Info(
						"no more announcements",
						"next_news_id",
						f.nextNewsID,
						"last_news_title",
						f.lastNewsTitle,
					)
					return nil
				}

				f.deps.Logger.Error(
					"failed to get announcement",
					"error",
					announcement.ErrorMessage,
				)

				return fmt.Errorf("failed to get announcement: %s", announcement.ErrorMessage)
			}

			f.nextNewsID = announcement.Data.ID + 1
			f.lastNewsTitle = announcement.Data.Title
		}
	}
}

func (f *AnnouncementByIDFetcher) initPoller() error {
	poller, err := httptools.NewProxyRotatingPollerBuilder().
		WithURL(fmt.Sprintf(f.deps.Config.UpbitAPI.AnnouncementByIDEndpoint, f.nextNewsID)).
		WithTargetRPS(f.deps.Config.ProxyRotatingPoller.TargetRPS).
		WithSingleProxyMaxRPS(f.deps.Config.UpbitAPI.AnnouncementByIDSingleIPMaxRPS).
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
