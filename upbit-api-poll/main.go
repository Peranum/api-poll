package main

import (
	"context"
	"flag"
	_ "net/http/pprof" // Import pprof for profiling
	"os"
	"os/signal"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/core"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di/setup"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	cfg := config.MustParseConfig(*configPath)
	deps := setup.MustContainer(cfg)
	deps.Logger.Info("Starting upbit api poller", "config", cfg.String())

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	run(ctx, deps)
}

func run(ctx context.Context, deps di.Container) {
	newsChan := streamNews(ctx, deps)

	monitor := core.NewNewsMonitor(newsChan, deps.Metrics)
	go monitor.StartMonitoring(ctx)

	go func() {
		deps.Logger.Info("Starting Prometheus metrics server on :8080")

		if err := deps.Metrics.StartMetricsServer("8080"); err != nil {
			deps.Logger.Error("Failed to start metrics server", "error", err)
		}
	}()

	<-ctx.Done()
}

func streamNews(ctx context.Context, deps di.Container) <-chan entity.NewsTitle {
	// announcementsFetcher, err := core.NewAnnouncementsFetcher(deps)
	// if err != nil {
	// 	panic(err)
	// }

	// announcementsNews, err := announcementsFetcher.StreamNewNewsTitles(ctx)
	// if err != nil {
	// 	panic(err)
	// }

	noticeByIDFetcher, err := core.NewNoticeByIDFetcher(deps)
	if err != nil {
		panic(err)
	}

	noticeNews, err := noticeByIDFetcher.StreamNewNoticeTitles(ctx)
	if err != nil {
		panic(err)
	}

	newsChan := make(chan entity.NewsTitle, 1024)

	go func() {
		var lastNews entity.NewsTitle

		for {
			select {
			case <-ctx.Done():
				return

			case news, ok := <-noticeNews:
				if !ok {
					continue
				}

				if news == lastNews {
					continue
				}

				deps.Logger.Info(
					"new notice title",
					"title",
					news,
					"from",
					"notice by id fetcher",
				)

				lastNews = news
				newsChan <- news

				// case news, ok := <-announcementsNews:
				// 	if !ok {
				// 		continue
				// 	}

				// 	if news == lastNews {
				// 		continue
				// 	}

				// 	deps.Logger.Info(
				// 		"new announcement title",
				// 		"title",
				// 		news,
				// 		"from",
				// 		"announcements fetcher",
				// 	)

				// 	lastNews = news
				// 	newsChan <- news
			}
		}
	}()

	return newsChan
}
