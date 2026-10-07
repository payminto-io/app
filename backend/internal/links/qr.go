package links

import (
	"fmt"
	"strings"

	"github.com/boombuler/barcode/qr"
)

// qrQuiet is the four-module quiet zone the QR specification requires around the symbol.
const qrQuiet = 4

// QRSVG renders the link URL as a QR code in SVG: one path, black modules on white, error correction M.
func QRSVG(url string) (string, error) {
	code, err := qr.Encode(url, qr.M, qr.Auto)
	if err != nil {
		return "", fmt.Errorf("links: encode QR: %w", err)
	}
	n := code.Bounds().Dx()
	size := n + 2*qrQuiet
	var path strings.Builder
	for y := range n {
		for x := range n {
			if r, _, _, _ := code.At(x, y).RGBA(); r == 0 {
				fmt.Fprintf(&path, "M%d %dh1v1h-1z", x+qrQuiet, y+qrQuiet)
			}
		}
	}
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges">`+
		`<rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="%s"/></svg>`, size, size, size, size, path.String()), nil
}
