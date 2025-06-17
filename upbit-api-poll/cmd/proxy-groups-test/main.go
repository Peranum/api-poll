package main

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/client/httpapi"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di/setup"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/fn"
)

func main() {
	cfg := config.MustParseConfig("configs/remote.yaml")
	deps := setup.MustContainer(cfg)

	proxies := fn.Map(cfg.APIPoller.Proxies, func(proxy entity.Proxy) *httpapi.UpbitClient {
		client, err := httpapi.NewUpbitClient(cfg.UpbitClient, &proxy, deps.Logger, deps.Metrics)
		if err != nil {
			panic(err)
		}

		return client
	})

	minExpectedProxiesCount := max(
		1,
		int(math.Ceil(cfg.APIPoller.TargetRPS/cfg.APIPoller.SingleProxyMaxRPS)),
	)
	possibleRPS := float64(len(proxies)) * cfg.APIPoller.SingleProxyMaxRPS
	if minExpectedProxiesCount > len(proxies) {
		panic(
			fmt.Sprintf(
				"not enough proxies for [tagrgetRPS=%f], because [minExpectedProxiesCount=%d], got [proxiesCount=%d], current [possibleRPS=%f]",
				cfg.APIPoller.TargetRPS,
				minExpectedProxiesCount,
				len(proxies),
				possibleRPS,
			),
		)
	}

	const proxyGroupsCount = 4
	groupExecTime := time.Duration(math.Ceil(1/cfg.APIPoller.SingleProxyMaxRPS)) * time.Second
	maxProxiesCountInGroup := int(math.Ceil(float64(len(proxies)) / float64(proxyGroupsCount)))
	pollingIntervalInGroup := time.Duration(float64(time.Second) / cfg.APIPoller.TargetRPS)
	proxyGroupRPS := float64(maxProxiesCountInGroup) / cfg.APIPoller.TargetRPS

	fmt.Println("Target RPS:", cfg.APIPoller.TargetRPS)
	fmt.Println("Possible RPS:", possibleRPS)
	fmt.Println("Single proxy max RPS:", cfg.APIPoller.SingleProxyMaxRPS)
	fmt.Println("Proxies count:", len(proxies))
	fmt.Println("Min expected proxies count:", minExpectedProxiesCount)
	fmt.Println("Proxy groups count:", proxyGroupsCount)
	fmt.Println("Max proxies count in group:", maxProxiesCountInGroup)
	fmt.Println("Single proxy RPS in group:", proxyGroupRPS)
	fmt.Println("Group exec time:", groupExecTime)
	fmt.Println("Polling interval in group:", pollingIntervalInGroup)

	proxyGroups := splitToGroups(proxies, proxyGroupsCount)

	ch := make(chan entity.UpbitResponse, 128)

	go func() {
		for i := 0; ; i = (i + 1) % len(proxyGroups) {
			proxyGroup := proxyGroups[i]
			startTime := time.Now()

			if len(proxyGroup) == 0 {
				fmt.Println("No proxies in group", i, "-> sleeping for", groupExecTime)
				time.Sleep(groupExecTime)
			}

			// Shuffle proxies in group to avoid situations when proxies at the start of the group are used more often
			// because of the way the group is selected
			rand.Shuffle(len(proxyGroup), func(i, j int) {
				proxyGroup[i], proxyGroup[j] = proxyGroup[j], proxyGroup[i]
			})

			for j := 0; ; j++ {
				elapsed := time.Since(startTime)
				if elapsed >= groupExecTime {
					fmt.Println("Group", i, "-> done")
					break
				}

				if j >= len(proxyGroup) {
					if j >= maxProxiesCountInGroup {
						j = -1
					} else {
						fmt.Println("Empty proxy", j, "-> sleeping for", pollingIntervalInGroup)
						time.Sleep(pollingIntervalInGroup)
					}

					continue
				}

				proxy := proxyGroup[j]

				go func(groupID, proxyID int, proxy *httpapi.UpbitClient) {
					resp, err := proxy.Request(context.Background())
					if err != nil {
						panic(err)
					}

					ch <- resp
				}(i, j, proxy)

				time.Sleep(pollingIntervalInGroup)
			}
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

func splitToGroups(proxies []*httpapi.UpbitClient, groupsCount int) [][]*httpapi.UpbitClient {
	groups := make([][]*httpapi.UpbitClient, groupsCount)

	for i := 0; i < len(proxies); i++ {
		groups[i%groupsCount] = append(groups[i%groupsCount], proxies[i])
	}

	return groups
}
