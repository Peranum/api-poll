package core

import "testing"

func TestContainsKoreanListingPattern(t *testing.T) {
	testCases := []struct {
		news    string
		expects bool
	}{
		{"라이브피어(LPT)(KRW, USDT 마켓), 포켓네트워크(POKT)(KRW 마켓) 디지털 자산 추가", true},
		{"소폰(SOPH) 신규 거래지원 안내 (KRW, BTC, USDT 마켓) (거래지원 개시 시점 및 매도 최저가 기준 가격 안내)", true},
		{"플록(FLOCK), 포르타(FORT) 신규 거래지원 안내 (BTC, USDT 마켓)", true},
		{"하이퍼레인(HYPER), 레드스톤(RED) 신규 거래지원 안내 (BTC, USDT 마켓)", true},
		{"쑨(SOON) 신규 거래지원 안내 (BTC, USDT 마켓) (거래지원 개시 시점 및 매도 최저가 기준 가격 안내)", true},
		{"커널다오(KERNEL) 신규 거래지원 안내 (BTC, USDT 마켓) (업비트 ATH 이벤트 안내)", true},
		{"펏지펭귄(PENGU) 신규 거래지원 안내 (KRW, BTC, USDT 마켓)", true},
		{"셀레스티아(TIA)(KRW, BTC, USDT 마켓), 아이오넷(IO)(BTC, USDT 마켓) 신규 거래지원 안내 (주문 타입 제한 관련 안내)", true},
		{"스톰엑스(STMX) 거래지원 종료 안내 (7/3 15:00)", false},
		{"넴(XEM) 거래지원 종료 안내 (7/3 15:00)", false},
		{"신세틱스(SNX) 거래 유의 종목 지정 기간 연장 안내", false},
		{"하이파이(HIFI) 거래지원 종료 안내 (5/12 17:00)", false},
		{"스톰엑스(STMX) 거래지원 종료 안내 (7/3 15:00)", false},
	}

	for i, tc := range testCases {
		if got := containsKoreanListingPattern(tc.news); got != tc.expects {
			t.Errorf("case %d failed: got %v, want %v, news: %s", i+1, got, tc.expects, tc.news)
		}
	}
}
