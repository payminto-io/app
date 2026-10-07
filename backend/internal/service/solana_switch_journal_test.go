package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/connectors"
	"github.com/payminto/payminto/backend/internal/connectors/chaindeposit"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/paymentswitch"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

// One finalized Solana deposit on a switch-owned payment yields exactly one payment journal, posted by
// the switch's chaindeposit path in USDC.SOLANA; the watcher, ledger wired, posts none (NEW2-M2).
func TestSolanaDeposit_SwitchPathPostsExactlyOnePaymentJournal(t *testing.T) {
	f := newSolanaFixture(t)
	must(t, paymentswitch.Migrate(f.db))
	// The watcher has its ledger wired as in production; it must skip the switch-owned payment.

	// Payminto side: an HD pool row for the owner so the switch's opener gets a real ATA.
	var family models.BlockchainFamily
	must(t, f.db.Where("code = ?", "sol").First(&family).Error)
	wallet := models.Wallet{Name: "HD Wallet (sol)", Kind: "hd", Status: "active", BlockchainFamilyID: family.ID, MemberID: f.member.ID}
	must(t, f.db.Create(&wallet).Error)
	must(t, f.db.Create(&models.AddressPool{Address: fxOwner, PathIndex: 0, Status: "available", WalletID: wallet.ID, BlockchainFamilyID: family.ID}).Error)
	poolSvc := NewAddressPoolService(repository.NewAddressPoolRepository(f.db), repository.NewWalletRepository(f.db), repository.NewBlockchainFamilyRepository(f.db), nil)
	addrSvc := NewDepositAddressService(repository.NewDepositAddressRepository(f.db), repository.NewWalletRepository(f.db), repository.NewBlockchainCurrencyRepository(f.db), nil, poolSvc).
		WithSolanaDepositAccounts(f.accounts, f.db, 0)
	paymentRepo := repository.NewPaymentRepository(f.db)
	payments := NewPaymentService(paymentRepo)
	payments.SetDepositAddressService(addrSvc)
	opener := func(amount decimal.Decimal, invoiceID, chainCode, currencyCode string, memberID, platformID uint) (chaindeposit.Opened, error) {
		res, err := payments.CreatePayment(CreatePaymentInput{AmountInUSD: amount, InvoiceID: &invoiceID, BlockchainCode: chainCode, CurrencyCode: currencyCode}, memberID, platformID)
		if err != nil {
			return chaindeposit.Opened{}, err
		}
		out := chaindeposit.Opened{Reference: res.Payment.ReferenceID, ExpiresAt: res.Payment.ExpiresAt}
		if res.DepositAddress != nil {
			out.Address = res.DepositAddress.Address
		}
		return out, nil
	}

	// Switch side: the chaindeposit connector over the Payminto backend, posting into the same ledger.
	reg := connectors.NewRegistry()
	must(t, reg.Register(chaindeposit.New(chaindeposit.NewPaymintoBackend(opener, paymentRepo, f.deposits, f.db))))
	selector := paymentswitch.FirstEnabledSelector{Merchants: paymentswitch.StaticMerchantConnectors{chaindeposit.Code}, Connectors: reg}
	sw := paymentswitch.New(f.db, reg, selector, f.journal)
	ctx := context.Background()
	merchant := strconv.FormatUint(uint64(f.member.ID), 10)
	intent, err := sw.Create(ctx, paymentswitch.CreateCommand{
		MerchantID: merchant, PlatformID: strconv.FormatUint(uint64(f.platform.ID), 10),
		Money:         paymentswitch.Money{Amount: decimal.NewFromInt(25), Asset: chaindeposit.PricingAsset},
		PaymentMethod: &connectors.PaymentMethod{Type: connectors.MethodChain, Token: "usdc", Details: map[string]string{chaindeposit.DetailChain: solana.ChainCode, chaindeposit.DetailAsset: "USDC"}},
		Confirm:       true,
	})
	must(t, err)
	var da models.DepositAddress
	must(t, f.db.Order("id DESC").First(&da).Error)
	if da.Address != fxUSDCATA {
		t.Fatalf("switch opened the payment on %s, want the owner's USDC ATA", da.Address)
	}

	// The watcher sees and finalizes the deposit without posting a journal.
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	if f.mustPoll(ctx) != 1 {
		t.Fatal("deposit not recorded")
	}
	f.statuses(map[string]any{"slot": 250000123, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	if f.mustConfirm(ctx) != 1 {
		t.Fatal("deposit not finalized")
	}
	var journals int64
	f.db.Table("ledger_journals").Where("kind = ?", "payment").Count(&journals)
	if journals != 0 {
		t.Fatalf("watcher posted %d journals for a switch-owned payment", journals)
	}

	// The switch syncs the attempt and posts the one payment journal.
	synced, err := sw.Sync(ctx, merchant, intent.ID)
	must(t, err)
	if synced.Status != paymentswitch.IntentSucceeded {
		t.Fatalf("intent status = %s, want succeeded", synced.Status)
	}
	f.db.Table("ledger_journals").Where("kind = ?", "payment").Count(&journals)
	if journals != 1 {
		t.Fatalf("payment journals = %d, want exactly one", journals)
	}
	var assets []string
	f.db.Table("ledger_lines").Joins("JOIN ledger_journals ON ledger_journals.id = ledger_lines.journal_id").
		Where("ledger_journals.kind = ?", "payment").Pluck("ledger_lines.asset", &assets)
	for _, a := range assets {
		if a != "USDC.SOLANA" {
			t.Fatalf("switch journal line in %s, want USDC.SOLANA", a)
		}
	}
	// A second sync and a second confirm round add nothing.
	if _, err := sw.Sync(ctx, merchant, intent.ID); err != nil {
		t.Fatal(err)
	}
	f.mustConfirm(ctx)
	f.db.Table("ledger_journals").Where("kind = ?", "payment").Count(&journals)
	if journals != 1 {
		t.Fatalf("payment journals after replay = %d", journals)
	}
}

// NEW2-M2: a payment created through the legacy POST /payment route (no switch attempt) gets exactly
// one payment journal from the watcher, with the switch tables present.
func TestSolanaDeposit_LegacyPaymentGetsExactlyOneWatcherJournal(t *testing.T) {
	f := newSolanaFixture(t)
	must(t, paymentswitch.Migrate(f.db))
	pr, _ := f.newPayment("25", f.usdc, fxOwner, fxUSDCATA)
	invoice := "merchant-invoice-42"
	must(t, f.db.Model(&models.PaymentRequest{}).Where("id = ?", pr.ID).Update("invoice_id", invoice).Error)
	f.script(map[string][]string{fxUSDCATA: {"usdc_transfer_checked.json"}})
	ctx := context.Background()
	if f.mustPoll(ctx) != 1 {
		t.Fatal("deposit not recorded")
	}
	f.statuses(map[string]any{"slot": 250000123, "confirmations": nil, "err": nil, "confirmationStatus": "finalized"})
	if f.mustConfirm(ctx) != 1 {
		t.Fatal("deposit not finalized")
	}
	var journals int64
	f.db.Table("ledger_journals").Where("kind = ?", "payment").Count(&journals)
	if journals != 1 {
		t.Fatalf("legacy payment journals = %d, want exactly one", journals)
	}
	f.mustConfirm(ctx)
	f.mustPoll(ctx)
	f.db.Table("ledger_journals").Where("kind = ?", "payment").Count(&journals)
	if journals != 1 {
		t.Fatalf("legacy payment journals after replay = %d", journals)
	}
}
