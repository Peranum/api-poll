package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/di/setup"
)

const endpointURL = "https://crix-static.upbit.com/crix_master"

func main() {
	cfg := config.MustParseConfig("/srv/configs/remote.yaml")
	deps := setup.MustContainer(cfg)

	transport := &http.Transport{
		MaxConnsPerHost:     100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     time.Second * 15,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		ForceAttemptHTTP2: true,
	}

	httpClient := &http.Client{
		Transport: transport,
	}

	refference, err := getLastCurrency(httpClient)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	deps.SendMessage(
		"<b>[crix-master-test]</b> Reference: <code>%s</code>",
		html.EscapeString(refference),
	)

	timer := time.NewTicker(time.Second * 5)
	defer timer.Stop()

	for range timer.C {
		current, err := getLastCurrency(httpClient)
		if err != nil {
			deps.Logger.Error("ERROR", "error", err)
			deps.SendMessage(
				"<b>[crix-master-test]</b> Error: <code>%s</code>",
				html.EscapeString(err.Error()),
			)
			continue
		}

		if current != refference {
			deps.SendMessage(
				"<b>[crix-master-test]</b> Current: <code>%s</code>",
				html.EscapeString(current),
			)
			deps.Logger.Info("NEW", "new_currency", current, "old_currency", refference)
			refference = current
		}
	}
}

func endpoint() string {
	timestamp := time.Now().UnixMilli()
	return fmt.Sprintf("%s?nonce=%d", endpointURL, timestamp)
}

func getLastCurrency(client *http.Client) (string, error) {
	resp, err := client.Get(endpoint())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var infos []MarketInfo
	if err := json.Unmarshal(body, &infos); err != nil {
		return "", err
	}

	lastInfo := infos[len(infos)-1]
	name := lastInfo.EnglishName

	return name, nil
}

type MarketInfo struct {
	Code                      string `json:"code"`
	LocalName                 string `json:"localName"`
	KoreanName                string `json:"koreanName"`
	EnglishName               string `json:"englishName"`
	Pair                      string `json:"pair"`
	BaseCurrencyCode          string `json:"baseCurrencyCode"`
	QuoteCurrencyCode         string `json:"quoteCurrencyCode"`
	Exchange                  string `json:"exchange"`
	MarketState               string `json:"marketState"`
	MarketStateForIOS         string `json:"marketStateForIOS"`
	IsTradingSuspended        bool   `json:"isTradingSuspended"`
	BaseCurrencyDecimalPlace  int    `json:"baseCurrencyDecimalPlace"`
	QuoteCurrencyDecimalPlace int    `json:"quoteCurrencyDecimalPlace"`
	Timestamp                 int64  `json:"timestamp"`
	TradeStatus               string `json:"tradeStatus"`
	OrderUnitVersion          string `json:"orderUnitVersion"`
	TradeSupportedMarket      bool   `json:"tradeSupportedMarket"`
}
