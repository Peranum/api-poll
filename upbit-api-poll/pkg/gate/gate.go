package gate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/service"
	"github.com/antihax/optional"
	gateapi "github.com/gateio/gateapi-go/v6"
)

const MetricGateOpenOrderDuration = "gate_open_order_duration_ms"

// OpenFuturesOrder открывает фьючерсный ордер на Gate.io
func OpenFuturesOrder(metrics service.MetricsService, token string, usdtAmount float64, leverage string) (map[string]interface{}, error) {
	// Используем StartTimer/ObserveDuration для метрики
	timer := metrics.StartTimer(MetricGateOpenOrderDuration)
	defer timer.ObserveDuration()
	apiKey := "a9b6cf044fba82cedadac3c9d494f395"
	apiSecret := "4fe9ebd4443ba1ef0660d35fd031ec3d2820137b27d9513faf411a9e3d6277f0"

	contract := token + "_USDT"

	cfg := gateapi.NewConfiguration()
	cfg.Key = apiKey
	cfg.Secret = apiSecret
	client := gateapi.NewAPIClient(cfg)
	ctx := context.Background()

	// 1. Установить левередж для позиции (максимально быстро, игнорируем ошибку десериализации)
	_, httpResp, _ := client.FuturesApi.UpdatePositionLeverage(ctx, "usdt", contract, leverage, nil)
	if httpResp != nil && httpResp.StatusCode >= 400 {
		return nil, fmt.Errorf("set leverage failed: %s", httpResp.Status)
	}

	// 2. Получить инфо о контракте и цене
	contractInfo, _, err := client.FuturesApi.GetFuturesContract(ctx, "usdt", contract)
	if err != nil {
		return nil, fmt.Errorf("get contract info: %w", err)
	}
	tickers, _, err := client.FuturesApi.ListFuturesTickers(ctx, "usdt", &gateapi.ListFuturesTickersOpts{Contract: optional.NewString(contract)})
	if err != nil || len(tickers) == 0 {
		return nil, fmt.Errorf("get ticker: %w", err)
	}
	currentPrice, _ := strconv.ParseFloat(tickers[0].Last, 64)
	quantoMultiplier, err := strconv.ParseFloat(contractInfo.QuantoMultiplier, 64)
	if err != nil {
		return nil, fmt.Errorf("parse quanto multiplier: %w", err)
	}

	// 3. Считаем количество контрактов, чтобы эквивалент было usdtAmount USDT
	// contractSize = usdtAmount / (currentPrice * quantoMultiplier)
	contractSizeF := usdtAmount / (currentPrice * quantoMultiplier)
	// Округляем до целого числа контрактов (Gate.io требует int64)
	contractSize := int64(contractSizeF)
	if contractSize == 0 && usdtAmount != 0 {
		// Если округление дало 0, но usdtAmount не 0, то ставим минимальный лот в нужную сторону
		if usdtAmount > 0 {
			contractSize = 1
		} else {
			contractSize = -1
		}
	}

	if contractInfo.OrderSizeMin > 0 && abs(contractSize) < contractInfo.OrderSizeMin {
		return nil, fmt.Errorf("size too small")
	}
	if contractInfo.OrderSizeMax > 0 && abs(contractSize) > contractInfo.OrderSizeMax {
		return nil, fmt.Errorf("size too large")
	}

	order := gateapi.FuturesOrder{
		Contract: contract,
		Size:     contractSize,
		Price:    "0",
		Tif:      "ioc",
	}

	result, _, err := client.FuturesApi.CreateFuturesOrder(ctx, "usdt", order, nil)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	return map[string]interface{}{
		"order": result,
		"conversion_info": map[string]interface{}{
			"usdt_amount":       usdtAmount,
			"contract_size":     contractSize,
			"quanto_multiplier": quantoMultiplier,
			"order_type":        "market",
			"current_price":     currentPrice,
		},
	}, nil
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
