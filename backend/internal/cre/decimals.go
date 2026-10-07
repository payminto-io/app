package cre

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

// Decimals maps an asset code (the currency part of a ledger asset) to its minor units.
// An asset with no entry is omitted from the liabilities snapshot rather than guessed.
type Decimals map[string]uint8

var defaultDecimals = Decimals{
	"USDC": 6, "USDT": 6, "PYUSD": 6, "EURC": 6, "TRX": 6,
	"BTC": 8, "SOL": 9,
	"ETH": 18, "DAI": 18, "POL": 18, "MATIC": 18, "BNB": 18,
	"USD": 2, "EUR": 2, "GBP": 2, "INR": 2, "SGD": 2, "AUD": 2, "CAD": 2, "CHF": 2, "AED": 2, "BRL": 2, "MXN": 2,
	"JPY": 0, "KRW": 0,
}

var decimalsCode = regexp.MustCompile(`^[A-Z0-9_-]{2,16}$`)

// ParseDecimals extends the built-in table with "CODE:n,..." from CRE_ASSET_DECIMALS.
func ParseDecimals(extra string) (Decimals, error) {
	out := make(Decimals, len(defaultDecimals))
	for k, v := range defaultDecimals {
		out[k] = v
	}
	for _, part := range strings.Split(extra, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		code, n, ok := strings.Cut(part, ":")
		code = strings.ToUpper(strings.TrimSpace(code))
		d, err := strconv.Atoi(strings.TrimSpace(n))
		if !ok || !decimalsCode.MatchString(code) || err != nil || d < 0 || d > 18 {
			return nil, fmt.Errorf("CRE_ASSET_DECIMALS entry %q must be CODE:decimals with decimals 0..18", part)
		}
		out[code] = uint8(d)
	}
	return out, nil
}

// Minor scales a ledger amount (major units, decimal string) to minor units for an asset; ok is false
// when the asset's decimals are unknown or the amount does not fit the grid.
func (d Decimals) Minor(asset, amount string) (*big.Int, uint8, bool) {
	code := asset
	if i := strings.LastIndex(asset, "."); i >= 0 {
		code = asset[:i]
	}
	dec, ok := d[code]
	if !ok {
		return nil, 0, false
	}
	v, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, 0, false
	}
	scaled := v.Shift(int32(dec))
	if !scaled.Equal(scaled.Truncate(0)) {
		return nil, 0, false
	}
	return scaled.BigInt(), dec, true
}

func parseBig(s string) (*big.Int, bool) {
	if s == "" {
		return nil, false
	}
	return new(big.Int).SetString(s, 10)
}
