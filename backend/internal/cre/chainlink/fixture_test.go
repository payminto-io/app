package chainlink

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/payminto/payminto/backend/internal/cre"
)

// The fixture is written by contracts/test/cre/GatewayAttestations.fixture.t.sol: two deliveries through
// Chainlink's real KeystoneForwarder into the audited contract, with the logs and calldata Foundry recorded.
type fixtureLog struct {
	Address     common.Address `json:"address"`
	TxHash      common.Hash    `json:"tx_hash"`
	BlockNumber uint64         `json:"block_number"`
	Index       uint           `json:"index"`
	Topics      []common.Hash  `json:"topics"`
	Data        string         `json:"data"`
}

type fixtureDelivery struct {
	Calldata   string      `json:"calldata"`
	Logs       string      `json:"logs"`
	ReportHash common.Hash `json:"report_hash"`
	TxHash     common.Hash `json:"tx_hash"`
}

type fixtureFile struct {
	Consumer      common.Address  `json:"consumer"`
	Forwarder     common.Address  `json:"forwarder"`
	WorkflowOwner common.Address  `json:"workflow_owner"`
	WorkflowID    common.Hash     `json:"workflow_id"`
	GatewayID     common.Hash     `json:"gateway_id"`
	Delivery1     fixtureDelivery `json:"delivery_1"`
	Delivery2     fixtureDelivery `json:"delivery_2"`
}

func loadFixture(t *testing.T) fixtureFile {
	t.Helper()
	raw, err := os.ReadFile("../../../../contracts/test/cre/fixtures/forwarder_logs.json")
	if err != nil {
		t.Fatalf("fixture missing; run `forge test --match-contract GatewayAttestationsFixtureTest` in contracts/: %v", err)
	}
	var f fixtureFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func (d fixtureDelivery) logs(t *testing.T) []Log {
	t.Helper()
	var entries []fixtureLog
	if err := json.Unmarshal([]byte(d.Logs), &entries); err != nil {
		t.Fatal(err)
	}
	out := make([]Log, 0, len(entries))
	for _, e := range entries {
		out = append(out, Log{Address: e.Address, TxHash: e.TxHash, BlockNumber: e.BlockNumber, Index: e.Index, Topics: e.Topics, Data: common.FromHex(e.Data)})
	}
	return out
}

func TestReaderAgainstRealForwarderDeliveries(t *testing.T) {
	fx := loadFixture(t)
	chain := &fakeChain{head: 2000, inputs: map[common.Hash][]byte{}}
	for _, d := range []fixtureDelivery{fx.Delivery1, fx.Delivery2} {
		chain.logs = append(chain.logs, d.logs(t)...)
		chain.inputs[d.TxHash] = common.FromHex(d.Calldata)
	}
	var gateway [32]byte
	copy(gateway[:], fx.GatewayID.Bytes())
	provider := New(Config{WorkflowIDs: map[cre.Kind][32]byte{cre.KindSolvency: fx.WorkflowID}, Consumer: fx.Consumer, GatewayID: gateway, ChunkBlocks: 10}, nil, chain)

	raws, next, err := provider.Poll(context.Background(), cre.KindSolvency, cre.Cursor{Block: 1000})
	if err != nil || len(raws) != 2 || next.Block != 2001 {
		t.Fatalf("raws = %d next %+v err %v", len(raws), next, err)
	}
	if raws[0].Evidence.ReportHash != [32]byte(fx.Delivery1.ReportHash) || raws[1].Evidence.ReportHash != [32]byte(fx.Delivery2.ReportHash) {
		t.Fatal("report hashes do not match what the contract logged")
	}
	for i, raw := range raws {
		if [32]byte(cre.PayloadHash(raw.Report)) != raw.Evidence.ReportHash {
			t.Fatalf("delivery %d: calldata slice does not hash to the logged reportHash", i+1)
		}
		if raw.Evidence.Emitter != [20]byte(fx.Consumer) || !raw.Evidence.Final {
			t.Fatalf("delivery %d evidence = %+v", i+1, raw.Evidence)
		}
	}
	for i, stored := range []bool{true, false} {
		if len(raws[i].Outcomes) != 2 || raws[i].Outcomes[0].Stored != stored || raws[i].Outcomes[1].Stored != stored {
			t.Fatalf("delivery %d outcomes = %+v, want both stored=%v", i+1, raws[i].Outcomes, stored)
		}
	}

	// The verifier, configured like a deployer would, attests delivery 1 and records delivery 2 as superseded.
	subjects := cre.NewMemoryStore()
	cp := cre.Checkpoint{ID: "cp-fixture", TakenAt: time.Unix(1_800_000_000, 0), Assets: []cre.AssetTotal{{Asset: "USDC.SOLANA", Liabilities: big.NewInt(1_000_000), Decimals: 6}, {Asset: "SOL", Liabilities: big.NewInt(7), Decimals: 9}}}
	cp.Hash = cre.SubjectKey("cp-fixture")
	_ = subjects.RememberSubjects(context.Background(), []cre.Subject{cre.CheckpointSubject(cp)})
	var owner [20]byte
	copy(owner[:], fx.WorkflowOwner.Bytes())
	v := &cre.Verifier{
		Provider: cre.ProviderChainlink, GatewayID: gateway, Consumer: fx.Consumer, Subjects: subjects, Chain: "fixture",
		Bindings: map[cre.Kind]cre.Binding{cre.KindSolvency: {ID: fx.WorkflowID, Owner: owner, Name: cre.KeystoneName("solvency")}},
		Now:      func() time.Time { return time.Unix(1_800_000_100, 0) },
	}
	rows, err := v.Verify(context.Background(), raws[0])
	if err != nil || len(rows) != 2 || rows[0].Status != cre.StatusAttested || rows[1].Status != cre.StatusAttested || rows[0].OnChain != cre.OnChainStored || rows[0].SubjectID != "cp-fixture" {
		t.Fatalf("delivery 1 rows = %+v err %v", rows, err)
	}
	rows, err = v.Verify(context.Background(), raws[1])
	if err != nil || len(rows) != 2 || rows[0].Status != cre.StatusIgnored || rows[1].Status != cre.StatusIgnored {
		t.Fatalf("delivery 2 rows = %+v err %v", rows, err)
	}
	// A wrong name binding (the audit's bug) refuses the real delivery.
	v.Bindings[cre.KindSolvency] = cre.Binding{ID: fx.WorkflowID, Owner: owner, Name: [10]byte{}}
	if _, err := v.Verify(context.Background(), raws[0]); err == nil {
		t.Fatal("wrong name binding accepted a real delivery")
	}
}
