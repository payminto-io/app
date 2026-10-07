package links

import (
	"strings"
	"testing"

	"github.com/boombuler/barcode/qr"
)

func TestQRSVGEncodesTheLinkURL(t *testing.T) {
	url := "https://checkout.test/l/AbCdEfGhIjKl"
	svg, err := QRSVG(url)
	if err != nil {
		t.Fatal(err)
	}
	code, _ := qr.Encode(url, qr.M, qr.Auto)
	n := code.Bounds().Dx()
	dark := 0
	for y := range n {
		for x := range n {
			if r, _, _, _ := code.At(x, y).RGBA(); r == 0 {
				dark++
			}
		}
	}
	if !strings.HasPrefix(svg, `<svg xmlns="http://www.w3.org/2000/svg"`) || strings.Count(svg, "h1v1h-1z") != dark || dark == 0 {
		t.Fatalf("svg has %d modules, symbol has %d", strings.Count(svg, "h1v1h-1z"), dark)
	}
	if strings.Contains(svg, "<script") {
		t.Fatal("svg carries script")
	}
}
