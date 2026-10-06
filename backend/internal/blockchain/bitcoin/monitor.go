package bitcoin

import (
	"context"
	"log"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/shopspring/decimal"
)

// startMonitor polls getblockcount, fetches new blocks, emits UTXO-like
// Transaction records on the channel. Cancel ctx to stop.
//
// NOTE: this is a minimal implementation — Phase F expands it with proper
// address matching and UTXO tracking.
func (a *Adapter) startMonitor(ctx context.Context, fromBlock uint64) (<-chan blockchain.Transaction, error) {
	ch := make(chan blockchain.Transaction, 100)

	if a.pool == nil {
		close(ch)
		return ch, nil
	}

	go func() {
		defer close(ch)
		current := fromBlock
		pollInterval := 30 * time.Second // BTC blocks are ~10 min apart

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(pollInterval):
			}

			var height uint64
			if err := a.call(ctx, "getblockcount", []any{}, &height); err != nil {
				log.Printf("[BTC] getblockcount error: %v", err)
				continue
			}

			if current == 0 {
				if height > 0 {
					current = height
				}
				continue
			}

			for current <= height {
				if ctx.Err() != nil {
					return
				}
				var hash string
				if err := a.call(ctx, "getblockhash", []any{current}, &hash); err != nil {
					log.Printf("[BTC] getblockhash(%d) error: %v", current, err)
					break
				}

				var block struct {
					Tx []struct {
						TxID string    `json:"txid"`
						Vout []btcVout `json:"vout"`
					} `json:"tx"`
				}
				if err := a.call(ctx, "getblock", []any{hash, 2}, &block); err != nil {
					log.Printf("[BTC] getblock(%s) error: %v", hash, err)
					break
				}

				for _, tx := range block.Tx {
					for _, vout := range tx.Vout {
						addr := vout.ScriptPubKey.Address
						if addr == "" && len(vout.ScriptPubKey.Addresses) > 0 {
							addr = vout.ScriptPubKey.Addresses[0]
						}
						if addr == "" {
							continue
						}
						// C2: parse via decimal to preserve precision.
						btc, err := decimal.NewFromString(vout.Value.String())
						if err != nil {
							continue
						}
						select {
						case <-ctx.Done():
							return
						case ch <- blockchain.Transaction{
							TxHash:      tx.TxID,
							ToAddress:   addr,
							Amount:      btc,
							Token:       "",
							BlockNumber: current,
							BlockHash:   hash,
						}:
						}
					}
				}
				current++
			}
		}
	}()

	return ch, nil
}
