package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	targetURL   = "https://api-manager.upbit.com/api/v1/announcements/5518"
	targetRPS   = 0.17
	maxRequests = 100 // Максимальное количество запросов для тестирования
)

var requestDelay time.Duration

func init() {
	// 1 секунда / 0.17 RPS = 5.882 секунды между запросами
	requestDelay = time.Duration(5882) * time.Millisecond
}

type RequestStats struct {
	TotalRequests int64
	SuccessCount  int64
	Error429Count int64
	StartTime     time.Time
	EndTime       time.Time
}

func main() {
	ctx := context.Background()

	log.Printf("Starting RPS test with target RPS: %.2f", targetRPS)
	log.Printf("Request delay: %v", requestDelay)
	log.Printf("Target URL: %s", targetURL)

	stats := &RequestStats{
		StartTime: time.Now(),
	}

	// Создаем HTTP клиент
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Канал для управления горутинами
	done := make(chan struct{})
	var wg sync.WaitGroup

	// Запускаем горутину для отправки запросов
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(done)

		ticker := time.NewTicker(requestDelay)
		defer ticker.Stop()

		for i := 0; i < maxRequests; i++ {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				wg.Add(1)
				go func(requestNum int) {
					defer wg.Done()
					makeRequest(client, stats, requestNum)
				}(i)
			}
		}
	}()

	// Ждем завершения всех запросов
	wg.Wait()
	stats.EndTime = time.Now()

	// Выводим статистику
	printStats(stats)
}

func makeRequest(client *http.Client, stats *RequestStats, requestNum int) {
	atomic.AddInt64(&stats.TotalRequests, 1)

	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		log.Printf("Request %d: Failed to create request: %v", requestNum, err)
		return
	}

	// Добавляем User-Agent для имитации браузера
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Request %d: Failed to execute request: %v", requestNum, err)
		return
	}
	defer resp.Body.Close()

	// Читаем тело ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Request %d: Failed to read response body: %v", requestNum, err)
	}

	if resp.StatusCode == http.StatusOK {
		atomic.AddInt64(&stats.SuccessCount, 1)
		log.Printf("Request %d: SUCCESS (200) - Body length: %d bytes", requestNum, len(body))
	} else if resp.StatusCode == http.StatusTooManyRequests {
		atomic.AddInt64(&stats.Error429Count, 1)
		log.Printf("Request %d: RATE LIMITED (429)", requestNum)
		log.Printf("Request %d: Response Headers:", requestNum)
		for name, values := range resp.Header {
			for _, value := range values {
				log.Printf("Request %d:   %s: %s", requestNum, name, value)
			}
		}
		log.Printf("Request %d: Response Body: %s", requestNum, string(body))
	} else {
		log.Printf("Request %d: HTTP %d - Body: %s", requestNum, resp.StatusCode, string(body))
	}
}

func printStats(stats *RequestStats) {
	duration := stats.EndTime.Sub(stats.StartTime)
	actualRPS := float64(stats.TotalRequests) / duration.Seconds()

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("RPS TEST RESULTS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Target RPS: %.2f\n", targetRPS)
	fmt.Printf("Actual RPS: %.4f\n", actualRPS)
	fmt.Printf("Total Requests: %d\n", stats.TotalRequests)
	fmt.Printf("Successful Requests (200): %d\n", stats.SuccessCount)
	fmt.Printf("Rate Limited Requests (429): %d\n", stats.Error429Count)
	fmt.Printf("Other Errors: %d\n", stats.TotalRequests-stats.SuccessCount-stats.Error429Count)
	fmt.Printf("Test Duration: %v\n", duration)
	fmt.Printf("Success Rate: %.2f%%\n", float64(stats.SuccessCount)/float64(stats.TotalRequests)*100)
	fmt.Printf("429 Error Rate: %.2f%%\n", float64(stats.Error429Count)/float64(stats.TotalRequests)*100)

	if stats.Error429Count > 0 {
		fmt.Println("\n" + strings.Repeat("!", 60))
		fmt.Println("RATE LIMITING DETECTED!")
		fmt.Printf("Received %d rate limit errors (429)\n", stats.Error429Count)
		fmt.Printf("Actual RPS maintained: %.4f\n", actualRPS)
		fmt.Println(strings.Repeat("!", 60))
	}
	fmt.Println(strings.Repeat("=", 60))
}
