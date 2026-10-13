// Command tempoprobe reads TransferWithMemo payments to an address on Tempo and prints them.
// Usage: go run ./cmd/tempoprobe -to 0x... [-from N] [-rpc URL]
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/payminto/payminto/backend/internal/blockchain/tempo"
)

func main() {
	rpc := flag.String("rpc", tempo.ModeratoRPC, "Tempo RPC URL")
	to := flag.String("to", "", "recipient address")
	back := flag.Uint64("back", 5000, "blocks back from head to scan")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	c, err := tempo.Dial(ctx, *rpc)
	if err != nil {
		log.Fatal(err)
	}
	head, err := c.Head(ctx)
	if err != nil {
		log.Fatal(err)
	}
	from := uint64(0)
	if head > *back {
		from = head - *back
	}
	transfers, err := c.MemoTransfersTo(ctx, []common.Address{tempo.PathUSD, tempo.AlphaUSD}, common.HexToAddress(*to), from, head)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("head=%d scanned=%d-%d found=%d\n", head, from, head, len(transfers))
	for _, t := range transfers {
		st, err := c.ReceiptStatus(ctx, t.TxHash)
		fmt.Printf("%s token=%s from=%s amount=%s memo=0x%x refs=%v status=%d err=%v\n  %s\n",
			t.Key(), t.Token.Hex(), t.From.Hex(), t.Amount, t.Memo, tempo.MemoCandidates(t.Memo), st, err, tempo.ExplorerTxURL(t.TxHash.Hex()))
	}
}
