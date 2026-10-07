package chainlink

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/payminto/payminto/backend/internal/cre"
)

// LogsForReport produces the logs GatewayAttestations emits for one accepted report, in the contract's order:
// one item event per item (SolvencyIgnored for the item indexes in ignoredItems), then ReportAccepted last. Tests and the
// local demo chain use it; nothing in the read path does.
func LogsForReport(consumer common.Address, meta cre.Metadata, report []byte, txHash common.Hash, block uint64, firstIndex uint, ignoredItems ...int) ([]Log, error) {
	contract, err := cre.ContractABI()
	if err != nil {
		return nil, err
	}
	decoded, err := cre.DecodeReport(report)
	if err != nil {
		return nil, err
	}
	skip := map[int]bool{}
	for _, i := range ignoredItems {
		skip[i] = true
	}
	var kindTopic common.Hash
	kindTopic[31] = decoded.Kind.Code()
	gateway := common.BytesToHash(decoded.GatewayID[:])
	observed := uint64(decoded.ObservedAt.Unix())
	index := firstIndex
	var logs []Log
	count := 0
	switch items := decoded.Items.(type) {
	case []cre.SolvencyItem:
		count = len(items)
		for i, it := range items {
			if skip[i] {
				data, err := contract.Events[EventSolvencyIgnored].Inputs.NonIndexed().Pack(observed, observed+1)
				if err != nil {
					return nil, err
				}
				logs = append(logs, Log{Address: consumer, TxHash: txHash, BlockNumber: block, Index: index, Topics: []common.Hash{contract.Events[EventSolvencyIgnored].ID, gateway, it.Asset}, Data: data})
				index++
				continue
			}
			data, err := contract.Events[EventSolvency].Inputs.NonIndexed().Pack(it.CheckpointHash, it.Liabilities, it.Reserves, it.Decimals, observed)
			if err != nil {
				return nil, err
			}
			logs = append(logs, Log{Address: consumer, TxHash: txHash, BlockNumber: block, Index: index, Topics: []common.Hash{contract.Events[EventSolvency].ID, gateway, it.Asset}, Data: data})
			index++
		}
	case []cre.DepositItem:
		count = len(items)
		for _, it := range items {
			var verdict common.Hash
			verdict[31] = it.Verdict
			data, err := contract.Events[EventDeposit].Inputs.NonIndexed().Pack(it.ChainID, it.TxRef, it.Token, it.Amount, it.Destination, it.SlotOrBlock, observed)
			if err != nil {
				return nil, err
			}
			logs = append(logs, Log{Address: consumer, TxHash: txHash, BlockNumber: block, Index: index, Topics: []common.Hash{contract.Events[EventDeposit].ID, gateway, it.DepositID, verdict}, Data: data})
			index++
		}
	case []cre.ConversionItem:
		count = len(items)
		for _, it := range items {
			data, err := contract.Events[EventConversion].Inputs.NonIndexed().Pack(it.ReferenceRate, it.ReferenceDecimals, it.DeviationBps, it.Feed, it.RoundID, observed)
			if err != nil {
				return nil, err
			}
			logs = append(logs, Log{Address: consumer, TxHash: txHash, BlockNumber: block, Index: index, Topics: []common.Hash{contract.Events[EventConversion].ID, gateway, it.ConversionID, it.Pair}, Data: data})
			index++
		}
	}
	accepted, err := contract.Events[EventReportAccepted].Inputs.NonIndexed().Pack(common.BytesToAddress(meta.Owner[:]), meta.WorkflowName, meta.ReportID, observed, big.NewInt(int64(count)), [32]byte(crypto.Keccak256(report)))
	if err != nil {
		return nil, fmt.Errorf("pack ReportAccepted: %w", err)
	}
	logs = append(logs, Log{Address: consumer, TxHash: txHash, BlockNumber: block, Index: index, Topics: []common.Hash{contract.Events[EventReportAccepted].ID, gateway, kindTopic, meta.WorkflowID}, Data: accepted})
	return logs, nil
}
