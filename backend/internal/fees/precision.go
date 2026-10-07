package fees

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// iso4217 lists active ISO 4217 codes; values are minor units (2 unless listed otherwise).
var iso4217 = func() map[string]int32 {
	m := map[string]int32{}
	for _, code := range strings.Fields(`AED AFN ALL AMD ANG AOA ARS AUD AWG AZN BAM BBD BDT BGN BMD BND BOB BOV BRL BSD
		BTN BWP BYN BZD CAD CDF CHE CHF CHW CNY COP COU CRC CUC CUP CVE CZK DKK DOP DZD EGP ERN ETB EUR FJD FKP GBP GEL
		GHS GIP GMD GTQ GYD HKD HNL HTG HUF IDR ILS INR IRR JMD KES KGS KHR KPW KYD KZT LAK LBP LKR LRD LSL MAD MDL MGA
		MKD MMK MNT MOP MRU MUR MVR MWK MXN MXV MYR MZN NAD NGN NIO NOK NPR NZD PAB PEN PGK PHP PKR PLN QAR RON RSD RUB
		SAR SBD SCR SDG SEK SGD SHP SLE SLL SOS SRD SSP STN SVC SYP SZL THB TJS TMT TOP TRY TTD TWD TZS UAH USD USN UYU
		UZS VED VES WST XCD YER ZAR ZMW ZWL`) {
		m[code] = 2
	}
	for _, code := range strings.Fields(`BIF CLP DJF GNF ISK JPY KMF KRW PYG RWF UGX UYI VND VUV XAF XOF XPF`) {
		m[code] = 0
	}
	for _, code := range strings.Fields(`BHD IQD JOD KWD LYD OMR TND`) {
		m[code] = 3
	}
	m["CLF"], m["UYW"] = 4, 4
	return m
}()

// defaultAssets are on-chain assets; precision is the token's decimals.
var defaultAssets = map[string]int32{
	"USDC": 6, "USDT": 6, "PYUSD": 6, "EURC": 6, "DAI": 18,
	"BTC": 8, "ETH": 18, "SOL": 9, "TRX": 6, "POL": 18, "MATIC": 18, "BNB": 18,
}

// Precision maps currency codes to minor units: ISO 4217 fiat plus a configurable set of on-chain assets.
type Precision struct {
	assets map[string]int32
}

func DefaultPrecision() Precision {
	assets := make(map[string]int32, len(defaultAssets))
	for k, v := range defaultAssets {
		assets[k] = v
	}
	return Precision{assets: assets}
}

var assetCode = regexp.MustCompile(`^[A-Z0-9]{2,10}$`)

// ParsePrecision reads FEES_ASSET_PRECISION: "CODE:decimals,..." added to the default on-chain assets.
func ParsePrecision(extra string) (Precision, error) {
	p := DefaultPrecision()
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return p, nil
	}
	for _, item := range strings.Split(extra, ",") {
		code, places, ok := strings.Cut(strings.TrimSpace(item), ":")
		code = strings.ToUpper(strings.TrimSpace(code))
		n, err := strconv.Atoi(strings.TrimSpace(places))
		switch {
		case !ok || err != nil || n < 0 || n > 18:
			return Precision{}, fmt.Errorf("fees: FEES_ASSET_PRECISION: %q must be CODE:decimals with decimals 0..18", item)
		case !assetCode.MatchString(code):
			return Precision{}, fmt.Errorf("fees: FEES_ASSET_PRECISION: invalid asset code %q", code)
		case isISO(code):
			return Precision{}, fmt.Errorf("fees: FEES_ASSET_PRECISION: %s is an ISO 4217 currency", code)
		}
		p.assets[code] = int32(n)
	}
	return p, nil
}

func isISO(code string) bool {
	_, ok := iso4217[code]
	return ok
}

func (p Precision) IsFiat(code string) bool { return isISO(code) }

// MinorUnits is the number of decimal places a currency settles in; unknown codes are refused.
func (p Precision) MinorUnits(code string) (int32, bool) {
	if n, ok := iso4217[code]; ok {
		return n, true
	}
	n, ok := p.assets[code]
	return n, ok
}

// fiatAsset is the ledger asset of a fiat currency: the bare ISO code, with no chain.
func fiatAsset(currency, chain string) (string, error) {
	if strings.TrimSpace(chain) != "" {
		return "", invalid("chain", "fiat currency %s has no chain", currency)
	}
	return currency, nil
}
