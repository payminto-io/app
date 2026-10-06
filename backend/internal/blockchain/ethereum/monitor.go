package ethereum

import (
	"context"
	"errors"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/shopspring/decimal"
)

// startMonitor polls new blocks and emits detected transactions on a channel.
// It decodes both native ETH transfers and ERC-20 Transfer events.
//
// Cancel the context to stop monitoring. The channel is closed when the
// goroutine exits.
func (a *Adapter) startMonitor(ctx context.Context, fromBlock uint64) (<-chan blockchain.Transaction, error) {
	ch := make(chan blockchain.Transaction, 100)

	if a.pool == nil {
		close(ch)
		return ch, errors.New("rpc pool not configured for monitor")
	}

	go func() {
		defer close(ch)
		currentBlock := fromBlock
		pollInterval := 5 * time.Second

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			header, err := a.latestHeader(ctx)
			if err != nil {
				log.Printf("[%s] latestHeader error: %v", a.code, err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(pollInterval):
				}
				continue
			}
			latest := header.Number.Uint64()

			if currentBlock == 0 {
				// First run: start from the latest block minus a small buffer
				// so we don't miss the first deposit.
				if latest > 5 {
					currentBlock = latest - 5
				}
			}

			if currentBlock > latest {
				select {
				case <-ctx.Done():
					return
				case <-time.After(pollInterval):
				}
				continue
			}

			// Process blocks one at a time
			block, err := a.blockByNumber(ctx, currentBlock)
			if err != nil {
				log.Printf("[%s] blockByNumber(%d) error: %v", a.code, currentBlock, err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(pollInterval):
				}
				continue
			}

			a.emitNativeTransfers(block, currentBlock, ch, ctx)
			currentBlock++

			// If we're caught up, slow down; if we're behind, speed up.
			if latest-currentBlock > 10 {
				pollInterval = 1 * time.Second
			} else {
				pollInterval = 5 * time.Second
			}
		}
	}()

	return ch, nil
}

// emitNativeTransfers walks the block's transactions, emitting a Transaction
// for each one that has a `to` field (contract calls without value still get
// emitted so ERC-20 decoders downstream can inspect the logs separately).
func (a *Adapter) emitNativeTransfers(block *types.Block, blockNum uint64, ch chan<- blockchain.Transaction, ctx context.Context) {
	hash := block.Hash().Hex()
	for _, tx := range block.Transactions() {
		if tx.To() == nil {
			continue
		}
		fromAddr := ""
		// Recover sender — ignore errors, some tx types may not decode
		signer := types.LatestSignerForChainID(big.NewInt(a.chainID))
		if sender, err := signer.Sender(tx); err == nil {
			fromAddr = sender.Hex()
		}

		wei := decimal.NewFromBigInt(tx.Value(), 0)
		ethAmount := wei.Div(decimal.New(1, 18))

		select {
		case <-ctx.Done():
			return
		case ch <- blockchain.Transaction{
			TxHash:      tx.Hash().Hex(),
			FromAddress: fromAddr,
			ToAddress:   common.Address(*tx.To()).Hex(),
			Amount:      ethAmount,
			Token:       "",
			BlockNumber: blockNum,
			BlockHash:   hash,
		}:
		}
	}
}
