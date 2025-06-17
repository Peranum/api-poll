package tickers

import (
	"regexp"
	"strings"
	"sync"
	"unicode"
)

var tickersPool = sync.Pool{
	New: func() interface{} {
		return make([]string, 0, 8)
	},
}

// ВАЖНО: функция предполагает, что входящее сообщение всегда начинается с "Market Support for"
func ExtractTickers(message string) []string {
	pos := len("Market Support for")
	message = message[pos:]

	for pos < len(message) && unicode.IsSpace(rune(message[pos])) {
		pos++
	}
	message = message[pos:]

	tickers := tickersPool.Get().([]string)
	tickers = tickers[:0]
	defer tickersPool.Put(tickers)

	end := 0
	inParentheses := false
	for i := 0; i < len(message); i++ {
		if message[i] == '(' {
			if !inParentheses {
				end = i
				break
			}
			inParentheses = true
		} else if message[i] == ')' {
			inParentheses = false
		}
	}
	if end == 0 {
		end = len(message)
	}

	start := 0
	inTicker := false
	tickerStart := 0
	tickerEnd := 0

	for i := 0; i < end; i++ {
		ch := message[i]

		switch ch {
		case '(':
			tickerStart = i + 1
			inTicker = true
		case ')':
			if inTicker {
				tickerEnd = i
				if tickerStart < tickerEnd {
					ticker := strings.TrimSpace(message[tickerStart:tickerEnd])
					if ticker != "" {
						tickers = append(tickers, ticker)
					}
				}
				inTicker = false
			}
		case ',':
			if !inTicker {
				part := strings.TrimSpace(message[start:i])
				if part != "" && !strings.Contains(part, "(") {
					tickers = append(tickers, part)
				}
				start = i + 1
			}
		}
	}

	if !inTicker && start < end {
		part := strings.TrimSpace(message[start:end])
		if part != "" && !strings.Contains(part, "(") {
			tickers = append(tickers, part)
		}
	}

	result := make([]string, len(tickers))
	copy(result, tickers)
	return result
}

// ExtractKoreanTickers извлекает тикеры из корейских новостей
func ExtractKoreanTickers(message string) []string {
	re := regexp.MustCompile(`\(([A-Z0-9]{2,10})\)`)
	matches := re.FindAllStringSubmatch(message, -1)

	tickers := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			tickers = append(tickers, match[1])
		}
	}
	return tickers
}
