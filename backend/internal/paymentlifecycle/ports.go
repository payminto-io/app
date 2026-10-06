package paymentlifecycle

import "context"

type PaymentLifecycle interface {
	Open(context.Context, OpenPayment) (PaymentView, error)
}

type QuoteAdapter interface {
	Quote(context.Context, QuoteRequest) (Quote, error)
}

// Store provides a receipt-first replay read followed, only on a miss, by one
// atomic Open operation. A successful Open return means the receipt, Invoice,
// Quote, Payment Method, address assignment, lifecycle event, and Outbox Event
// committed together, or a concurrent identical Open replayed.
type Store interface {
	// LookupReceipt performs the receipt-first read needed for replay without
	// calling a Quote Adapter. It returns conflict when the scoped key exists
	// with a different stable request hash.
	LookupReceipt(context.Context, ReceiptLookup) (ReceiptResult, error)
	Open(context.Context, OpenRecord) (StoreResult, error)
}
