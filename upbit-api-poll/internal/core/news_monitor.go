package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/entity"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/service"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/tickers"
	gate "github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/gate"
)

type NewsMonitor struct {
	newsChan <-chan entity.NewsTitle
	metrics  *service.PrometheusService
}

var koreanListingPatterns = []string{
	"신규 거래지원 안내",
	"디지털 자산 추가",
	"상장 안내",
}

func containsKoreanListingPattern(news string) bool {
	for _, pattern := range koreanListingPatterns {
		if strings.Contains(news, pattern) {
			return true
		}
	}
	return false
}

func NewNewsMonitor(
	newsChan <-chan entity.NewsTitle,
	metrics *service.PrometheusService,
) *NewsMonitor {
	return &NewsMonitor{
		newsChan: newsChan,
		metrics:  metrics,
	}
}

func (m *NewsMonitor) StartMonitoring(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Background monitoring stopped")
			return
		case news := <-m.newsChan:
			fmt.Printf("Received news: %s\n", news)

			if strings.Contains(news, "Market Support for") ||
				containsKoreanListingPattern(news) {
				fmt.Printf("!!! FOUND LISTING NEWS: %s !!!\n", news)

				extractedTickers := tickers.ExtractKoreanTickers(news)
				if len(extractedTickers) > 0 {
					ticker := extractedTickers[0]
					go func(ticker string) {
						_, err := gate.OpenFuturesOrder(m.metrics, ticker, 10.0, "20")
						if err != nil {
							fmt.Printf("Failed to open order for %s: %v\n", ticker, err)
						} else {
							fmt.Printf("Order opened for %s\n", ticker)
						}
					}(ticker)
				}
			}
		}
	}
}
