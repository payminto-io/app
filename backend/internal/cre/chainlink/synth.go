package chainlink

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/payminto/payminto/backend/internal/cre"
)

// LogsForReport produces the logs GatewayAttestations emits for one accepted report: ReportAccepted
// then one item event per item, in that order. Tests and the local demo chain use it; nothing in the
// read path does, so the reader is only ever exercised against event-shaped data.
func LogsForReport(consumer common.Address, meta cre.Metadata, report []byte, txHash common.Hash, block uint64, firstIndex uint) ([]Log, error) {
	contract, err := cre.ContractABI()
	if err != nil {
		return nil, err
	}
	decoded, err := cre.DecodeReport(report)
	if err != nil {
		return nil, err
	}
	var kindTopic common.Hash
	kindTopic[31] = decoded.Kind.Code()
	gateway := common.BytesToHash(decoded.GatewayID[:])
	observed := uint64(decoded.ObservedAt.Unix())
	count := 0
	switch items := decoded.Items.(type) {
	case []cre.SolvencyItem:
		count = len(items)
	case []cre.DepositItem:
		count = len(items)
	case []cre.ConversionItem:
		count = len(items)
	}
	accepted, err := contract.Events[EventReportAccepted].Inputs.NonIndexed().Pack(common.BytesToAddress(meta.Owner[:]), meta.WorkflowName, meta.ReportID, observed, big.NewInt(int64(count)), [32]byte(crypto.Keccak256(report)))
	if err != nil {
		return nil, fmt.Errorf("pack ReportAccepted: %w", err)
	}
	index := firstIndex
	logs := []Log{{Address: consumer, TxHash: txHash, BlockNumber: block, Index: index, Topics: []common.Hash{contract.Events[EventReportAccepted].ID, gateway, kindTopic, meta.WorkflowID}, Data: accepted}}
	switch items := decoded.Items.(type) {
	case []cre.SolvencyItem:
		for _, it := range items {
			index++
			data, err := contract.Events[EventSolvency].Inputs.NonIndexed().Pack(it.CheckpointHash, it.Liabilities, it.Reserves, it.Decimals, observed)
			if err != nil {
				return nil, err
			}
			logs = append(logs, Log{Address: consumer, TxHash: txHash, BlockNumber: block, Index: index, Topics: []common.Hash{contract.Events[EventSolvency].ID, gateway, it.Asset}, Data: data})
		}
	case []cre.DepositItem:
		for _, it := range items {
			index++
			var verdict common.Hash
			verdict[31] = it.Verdict
			data, err := contract.Events[EventDeposit].Inputs.NonIndexed().Pack(it.ChainID, it.TxRef, it.Token, it.Amount, it.Destination, it.SlotOrBlock, observed)
			if err != nil {
				return nil, err
			}
			logs = append(logs, Log{Address: consumer, TxHash: txHash, BlockNumber: block, Index: index, Topics: []common.Hash{contract.Events[EventDeposit].ID, gateway, it.DepositID, verdict}, Data: data})
		}
	case []cre.ConversionItem:
		for _, it := range items {
			index++
			data, err := contract.Events[EventConversion].Inputs.NonIndexed().Pack(it.ReferenceRate, it.ReferenceDecimals, it.DeviationBps, it.Feed, it.RoundID, observed)
			if err != nil {
				return nil, err
			}
			logs = append(logs, Log{Address: consumer, TxHash: txHash, BlockNumber: block, Index: index, Topics: []common.Hash{contract.Events[EventConversion].ID, gateway, it.ConversionID, it.Pair}, Data: data})
		}
	}
	return logs, nil
}
