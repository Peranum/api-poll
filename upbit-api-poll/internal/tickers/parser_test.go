package tickers

import (
	"testing"
)

func TestExtractTickers(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected []string
	}{
		{
			name:     "Single ticker with markets",
			message:  "Market Support for Sign(SIGN) (KRW, BTC, USDT Market)",
			expected: []string{"SIGN"},
		},
		{
			name:     "Multiple tickers without parentheses",
			message:  "Market Support for ACS, GO, OBSR, QTCON, RLY (USDT Market)",
			expected: []string{"ACS", "GO", "OBSR", "QTCON", "RLY"},
		},
		{
			name:     "Two tickers with parentheses",
			message:  "Market Support for Hyperlane(HYPER), RedStone(RED) (BTC, USDT Market)",
			expected: []string{"HYPER", "RED"},
		},
		{
			name:     "Multiple markets per ticker",
			message:  "Market Support for Celestia(TIA)(KRW, BTC, USDT market), io.net(IO)(BTC, USDT market)",
			expected: []string{"TIA", "IO"},
		},
		{
			name:     "Mixed format tickers",
			message:  "Market Support for BTC, ETH(ETH), Ripple(XRP), DOGE (Multi Market)",
			expected: []string{"BTC", "ETH", "XRP", "DOGE"},
		},
		// Пограничные случаи
		{
			name:     "Single ticker no markets",
			message:  "Market Support for MyShell(SHELL)",
			expected: []string{"SHELL"},
		},
		{
			name:     "Extra spaces between tickers",
			message:  "Market Support for   Token1(TKN1),    Token2(TKN2)   (Markets)",
			expected: []string{"TKN1", "TKN2"},
		},
		{
			name:     "No space after prefix",
			message:  "Market Support forToken(TKN) (Markets)",
			expected: []string{"TKN"},
		},
		// Длинные сообщения
		{
			name: "Many tickers",
			message: "Market Support for Token1(TKN1), Token2(TKN2), Token3(TKN3), Token4(TKN4), " +
				"Token5(TKN5), Token6(TKN6), Token7(TKN7), Token8(TKN8), Token9(TKN9), Token10(TKN10) (Markets)",
			expected: []string{
				"TKN1",
				"TKN2",
				"TKN3",
				"TKN4",
				"TKN5",
				"TKN6",
				"TKN7",
				"TKN8",
				"TKN9",
				"TKN10",
			},
		},
		{
			name: "Long token names",
			message: "Market Support for VeryLongTokenName1(VLTN1), VeryLongTokenName2(VLTN2), " +
				"VeryLongTokenName3(VLTN3) (Markets)",
			expected: []string{"VLTN1", "VLTN2", "VLTN3"},
		},
		{
			name:     "Nested parentheses in name",
			message:  "Market Support for Token(One)(TKN1), Token(Two)(TKN2) (Markets)",
			expected: []string{"TKN1", "TKN2"},
		},
		{
			name:     "Special characters in names",
			message:  "Market Support for Token.IO(TKN1), Token-X(TKN2), Token_Y(TKN3) (Markets)",
			expected: []string{"TKN1", "TKN2", "TKN3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTickers(tt.message)
			if !equalSlices(got, tt.expected) {
				t.Errorf("ExtractTickers() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}


func TestExtractKoreanTickers(t *testing.T) {
	testCases := []struct {
		name     string
		message  string
		expected []string
	}{
		{
			name:     "Несколько тикеров с рынками",
			message:  "라이브피어(LPT)(KRW, USDT 마켓), 포켓네트워크(POKT)(KRW 마켓) 디지털 자산 추가",
			expected: []string{"LPT", "POKT"},
		},
		{
			name:     "Один тикер",
			message:  "소폰(SOPH) 신규 거래지원 안내 (KRW, BTC, USDT 마켓) (거래지원 개시 시점 및 매도 최저가 기준 가격 안내)",
			expected: []string{"SOPH"},
		},
		{
			name:     "Два тикера",
			message:  "플록(FLOCK), 포르타(FORT) 신규 거래지원 안내 (BTC, USDT 마켓)",
			expected: []string{"FLOCK", "FORT"},
		},
		{
			name:     "Два тикера с рынками",
			message:  "하이퍼레인(HYPER), 레드스톤(RED) 신규 거래지원 안내 (BTC, USDT 마켓)",
			expected: []string{"HYPER", "RED"},
		},
		{
			name:     "Один тикер с длинным сообщением",
			message:  "쑨(SOON) 신규 거래지원 안내 (BTC, USDT 마켓) (거래지원 개시 시점 및 매도 최저가 기준 가격 안내)",
			expected: []string{"SOON"},
		},
		{
			name:     "Один тикер с событием",
			message:  "커널다오(KERNEL) 신규 거래지원 안내 (BTC, USDT 마켓) (업비트 ATH 이벤트 안내)",
			expected: []string{"KERNEL"},
		},
		{
			name:     "Один тикер с тремя рынками",
			message:  "펏지펭귄(PENGU) 신규 거래지원 안내 (KRW, BTC, USDT 마켓)",
			expected: []string{"PENGU"},
		},
		{
			name:     "Два тикера с разными рынками",
			message:  "셀레스티아(TIA)(KRW, BTC, USDT 마켓), 아이오넷(IO)(BTC, USDT 마켓) 신규 거래지원 안내 (주문 타입 제한 관련 안내)",
			expected: []string{"TIA", "IO"},
		},
		{
			name:     "Два тикера без слова '마켓' в рынках",
			message:  "셀레스티아(TIA)(KRW, BTC, USDT), 아이오넷(IO)(BTC, USDT) 신규 거래지원 안내 (주문 타입 제한 관련 안내)",
			expected: []string{"TIA", "IO"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractKoreanTickers(tc.message)
			if !equalSlices(got, tc.expected) {
				t.Errorf(
					"ExtractKoreanTickers() = %v, want %v, message: %s",
					got,
					tc.expected,
					tc.message,
				)
			}
		})
	}
}
