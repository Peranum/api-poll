package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/internal/config"
	"github.com/Shadow-Web3-development-studio/listings/upbit-api-poll/pkg/httptools"
)

func main() {
	configPathFlag := flag.String("config", "", "path to the config file")
	outputPathFlag := flag.String("output", "", "path to the output file")
	flag.Parse()

	var output io.Writer

	if *outputPathFlag != "" {
		outputFile, err := os.Create(*outputPathFlag)
		if err != nil {
			log.Fatalf("[ERROR] failed to create output file: %v", err)
		}

		defer outputFile.Close()
		output = outputFile
	} else {
		output = os.Stdout
	}

	if *configPathFlag == "" {
		log.Fatal("[ERROR] config path is required")
	}

	cfg := config.MustParseConfig(*configPathFlag)

	proxies := cfg.ProxyRotatingPoller.Proxies
	clearedProxies := make([]httptools.Proxy, 0, len(proxies))

	proxyMap := make(map[httptools.Proxy]bool)

	count := 0

	for _, proxy := range proxies {
		if !proxyMap[proxy] {
			clearedProxies = append(clearedProxies, proxy)
		}

		proxyMap[proxy] = true
		count++
	}

	for _, proxy := range clearedProxies {
		fmt.Fprintln(
			output,
			strings.Join(
				[]string{proxy.Host, strconv.Itoa(proxy.Port), proxy.Username, proxy.Password},
				":",
			),
		)
	}

	log.Printf("[INFO] total proxies: %d, cleared proxies: %d", count, len(clearedProxies))
}
