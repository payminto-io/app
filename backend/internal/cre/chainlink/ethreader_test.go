package chainlink

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type fakeEVM struct {
	finalized    *big.Int
	finalizedErr error
	latest       uint64
}

func (f fakeEVM) HeaderByNumber(context.Context, *big.Int) (*types.Header, error) {
	if f.finalizedErr != nil {
		return nil, f.finalizedErr
	}
	return &types.Header{Number: f.finalized}, nil
}
func (f fakeEVM) BlockNumber(context.Context) (uint64, error) { return f.latest, nil }
func (fakeEVM) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}
func (fakeEVM) TransactionByHash(context.Context, common.Hash) (*types.Transaction, bool, error) {
	return nil, false, errors.New("unused")
}

func TestFinalizedHeadPrefersTheTagAndNeverFallsBackToZeroConfirmations(t *testing.T) {
	ctx := context.Background()
	if h, err := NewReaderWith(fakeEVM{finalized: big.NewInt(500), latest: 600}, 0).FinalizedHead(ctx); err != nil || h != 500 {
		t.Fatalf("tag: %d %v", h, err)
	}
	noTag := fakeEVM{finalizedErr: errors.New("method not found"), latest: 600}
	if _, err := NewReaderWith(noTag, 0).FinalizedHead(ctx); !errors.Is(err, ErrNoFinality) {
		t.Fatalf("zero confirmations without the tag must refuse, got %v", err)
	}
	if h, err := NewReaderWith(noTag, 12).FinalizedHead(ctx); err != nil || h != 588 {
		t.Fatalf("fallback: %d %v", h, err)
	}
	if h, err := NewReaderWith(fakeEVM{finalizedErr: errors.New("x"), latest: 5}, 12).FinalizedHead(ctx); err != nil || h != 0 {
		t.Fatalf("young chain: %d %v", h, err)
	}
}
