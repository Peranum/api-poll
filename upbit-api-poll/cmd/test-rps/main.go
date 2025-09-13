package main

import (
	"context"
	"log"
	"time"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/httptools"
)

func main() {
	client, err := httptools.NewClientHTTP2()
	if err != nil {
		log.Fatal(err)
	}

	currentRPS := 0.2
	delta := 0.1
	firstKnock := true
	pollingInterval := time.Duration(float64(time.Second) / currentRPS)

	startTime := time.Now()

	log.Printf(
		"Polling started at %s [interval: %s] [rps: %f]",
		startTime.Format(time.RFC3339),
		pollingInterval,
		currentRPS,
	)

	for {
		resp, err := client.Request(
			context.Background(),
			"https://api-manager.upbit.com/api/v1/announcements/5569",
		)
		if err != nil {
			log.Println("Error:", err)
			continue
		}

		if resp.IsTooManyRequests() {
			log.Printf(
				"Too many requests after %s of polling on polling interval %s [rps: %f]",
				time.Since(startTime),
				pollingInterval,
				currentRPS,
			)
			if firstKnock {
				delta /= 2
			}

			firstKnock = false
			currentRPS -= delta
			pollingInterval = time.Duration(float64(time.Second) / currentRPS)
			startTime = time.Now()

			log.Printf(
				"Polling interval decreased to %s [rps: %f]",
				pollingInterval,
				currentRPS,
			)
			log.Printf("Sleeping for 1 hour")
			time.Sleep(1 * time.Hour)
			log.Printf(
				"Polling started at %s [interval: %s] [rps: %f]",
				startTime.Format(time.RFC3339),
				pollingInterval,
				currentRPS,
			)
			continue
		}

		if time.Since(startTime) > time.Hour {
			firstKnock = true
			currentRPS += delta
			pollingInterval = time.Duration(float64(time.Second) / currentRPS)
			startTime = time.Now()

			log.Printf("Polling interval increased to %s [rps: %f]", pollingInterval, currentRPS)
		}

		time.Sleep(pollingInterval)
	}
}
