package main

import (
	"context"
	"fmt"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/client/httpapi"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di/setup"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
)

func main() {
	cfg := config.MustParseConfig("configs/remote.yaml")
	deps := setup.MustContainer(cfg)

	cfg.APIPoller.TargetRPS = 1

	proxies := []*httpapi.UpbitClient{func() *httpapi.UpbitClient {
		client, err := httpapi.NewUpbitClient(cfg.UpbitClient, nil, deps.Logger, deps.Metrics)
		if err != nil {
			panic(err)
		}
		return client
	}()}

	pollingIntervalInGroup := time.Duration(float64(time.Second) / cfg.APIPoller.TargetRPS)

	fmt.Println("Target RPS:", cfg.APIPoller.TargetRPS)
	fmt.Println("Single proxy max RPS:", cfg.APIPoller.SingleProxyMaxRPS)
	fmt.Println("Proxies count:", len(proxies))
	fmt.Println("Polling interval in group:", pollingIntervalInGroup)

	ch := make(chan entity.UpbitResponse, 128)

	go func() {
		for j := 0; ; j++ {
			proxy := proxies[j%len(proxies)]

			go func(proxy *httpapi.UpbitClient) {
				resp, err := proxy.Request(context.Background())
				if err != nil {
					panic(err)
				}

				ch <- resp
			}(proxy)

			time.Sleep(pollingIntervalInGroup)
		}

	}()

	for msg := range ch {
		fmt.Printf("Response\n\tID: %v\n\tCreatedAt: %s\n\tStatus: %d\n\tProxyAddr: %s\n\tHeaders: %v\n\tBody: %s\n",
			msg.ID,
			msg.CreatedAt,
			msg.Status,
			msg.ProxyAddr,
			msg.Headers,
			msg.Body)
	}
}
