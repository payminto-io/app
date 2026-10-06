package tron

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/shopspring/decimal"
)

// startMonitor polls getnowblock + getblockbynum for new Tron blocks and
// emits transfers on a channel. Cancel ctx to stop.
func (a *Adapter) startMonitor(ctx context.Context, fromBlock uint64) (<-chan blockchain.Transaction, error) {
	ch := make(chan blockchain.Transaction, 100)

	if a.pool == nil {
		close(ch)
		return ch, nil
	}

	go func() {
		defer close(ch)
		current := fromBlock
		pollInterval := 3 * time.Second // Tron blocks are ~3s

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(pollInterval):
			}

			var latest struct {
				BlockHeader struct {
					RawData struct {
						Number int64 `json:"number"`
					} `json:"raw_data"`
				} `json:"block_header"`
			}
			if err := a.post(ctx, "/wallet/getnowblock", nil, &latest); err != nil {
				log.Printf("[TRX] getnowblock error: %v", err)
				continue
			}

			latestNum := uint64(latest.BlockHeader.RawData.Number)
			if current == 0 {
				current = latestNum
				continue
			}

			for current <= latestNum {
				if ctx.Err() != nil {
					return
				}
				var block struct {
					BlockID      string `json:"blockID"`
					Transactions []struct {
						TxID    string `json:"txID"`
						RawData struct {
							Contract []tronContract `json:"contract"`
						} `json:"raw_data"`
					} `json:"transactions"`
				}
				if err := a.post(ctx, "/wallet/getblockbynum", map[string]any{"num": current}, &block); err != nil {
					log.Printf("[TRX] getblockbynum(%d) error: %v", current, err)
					break
				}

				for _, tx := range block.Transactions {
					for _, contract := range tx.RawData.Contract {
						if contract.Type != "TransferContract" {
							continue
						}
						to := contract.Parameter.Value.ToAddress
						from := contract.Parameter.Value.OwnerAddress

						// C1: parse amount via json.Number to avoid float64 precision loss.
						var amountNum json.Number
						if err := json.Unmarshal(contract.Parameter.Value.Amount, &amountNum); err != nil {
							continue
						}
						amountDec, err := decimal.NewFromString(amountNum.String())
						if err != nil {
							continue
						}
						// TRX amounts in sun (10^6 per TRX).
						trxAmount := amountDec.Div(decimal.New(1, 6))

						select {
						case <-ctx.Done():
							return
						case ch <- blockchain.Transaction{
							TxHash:      tx.TxID,
							FromAddress: from,
							ToAddress:   to,
							Amount:      trxAmount,
							Token:       "",
							BlockNumber: current,
							BlockHash:   block.BlockID,
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
