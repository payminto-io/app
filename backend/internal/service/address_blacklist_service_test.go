package service

import "testing"

func TestAddressBlacklist_EmptyWhenUnloaded(t *testing.T) {
	s := NewAddressBlacklistService(nil)
	if s.IsBlacklisted("ETH", "0xabc") {
		t.Error("empty blacklist should not match")
	}
}

func TestAddressBlacklist_LoadAndMatch(t *testing.T) {
	s := NewAddressBlacklistService(nil)
	s.entries[key("ETH", "0xBAD")] = struct{}{}
	s.entries[key("btc", "bc1qBlocked")] = struct{}{}

	cases := []struct {
		chain, addr string
		want        bool
	}{
		{"ETH", "0xbad", true},
		{"eth", "0xBAD", true},
		{"BTC", "bc1qblocked", true},
		{"ETH", "0xgood", false},
		{"TRX", "0xbad", false},
	}
	for _, tc := range cases {
		if got := s.IsBlacklisted(tc.chain, tc.addr); got != tc.want {
			t.Errorf("IsBlacklisted(%s,%s)=%v want %v", tc.chain, tc.addr, got, tc.want)
		}
	}
}
