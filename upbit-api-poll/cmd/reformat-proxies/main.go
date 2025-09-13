package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

func main() {
	filePathFlag := flag.String("path", "", "path to the file containing the proxies")
	outputPathFlag := flag.String(
		"output",
		"",
		"path to the file to write the formatted proxies to",
	)
	prefixFlag := flag.String("prefix", "", "prefix to add to the formatted proxies")
	flag.Parse()

	if *filePathFlag == "" {
		log.Fatal("[ERROR] path to the file containing the proxies is required")
	}

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

	prefix := *prefixFlag

	proxies, err := os.ReadFile(*filePathFlag)
	if err != nil {
		log.Fatalf("[ERROR] failed to read file: %v", err)
	}

	lines := strings.Split(string(proxies), "\n")

	successCount := 0
	failCount := 0

	fmt.Fprintln(output, prefix+"proxies:")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.Split(line, ":")
		if len(parts) != 4 {
			log.Printf("[WARN] invalid proxy format on line %d: %s", i, line)
			failCount++
			continue
		}

		fmt.Fprintln(
			output,
			prefix+prefix+"- { host: "+`"`+parts[0]+`"`+", port: "+parts[1]+", username: "+`"`+parts[2]+`"`+", password: "+`"`+parts[3]+`"`+" }",
		)
		successCount++
	}

	log.Printf(
		"[INFO] success: %d, fail: %d, total: %d",
		successCount,
		failCount,
		successCount+failCount,
	)
}
