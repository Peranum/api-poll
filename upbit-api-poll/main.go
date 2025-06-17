package main

import (
	"context"
	"flag"
	_ "net/http/pprof" // Import pprof for profiling
	"os"
	"os/signal"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/client/grpc"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/core"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di/setup"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/channel"
)

func main() {
	configPath := flag.String("config", "configs/local.yaml", "path to config file")
	flag.Parse()

	cfg := config.MustParseConfig(*configPath)
	deps := setup.MustContainer(cfg)
	deps.Logger.Info("Starting upbit api poller", "config", cfg)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	run(ctx, deps)
}

func run(ctx context.Context, deps di.Container) {
	poller, err := core.NewAPIPoller(deps)
	if err != nil {
		deps.Logger.Error("failed to create api poller", "error", err)
		return
	}

	gateClient, err := grpc.NewGateClient(deps.Config.GRPC)
	if err != nil {
		deps.Logger.Error("failed to create gate gRPC client", "error", err)
		return
	}
	defer gateClient.Close()

	newsChan := poller.StreamNews(ctx)
	newsBroadcast := channel.NewBroadcastAdapter(newsChan)
	defer newsBroadcast.Close()

	monitorChan, err := newsBroadcast.Follow()
	if err != nil {
		deps.Logger.Error("failed to create news broadcast", "error", err)
		return
	}
	defer newsBroadcast.Unfollow(monitorChan)

	monitor := core.NewNewsMonitor(gateClient, monitorChan, deps.Metrics)
	go monitor.StartMonitoring(ctx)

	go func() {

		deps.Logger.Info("Starting Prometheus metrics server on :8080")
		if err := deps.Metrics.StartMetricsServer("8080"); err != nil {
			deps.Logger.Error("Failed to start metrics server", "error", err)
		}
	}()

	<-ctx.Done()
}
