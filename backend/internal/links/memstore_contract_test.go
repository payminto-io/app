package links_test

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/links"
	"github.com/payminto/payminto/backend/internal/links/storetest"
)

func TestMemStoreContract(t *testing.T) {
	storetest.Run(t, func(t *testing.T) storetest.Fixture {
		s := links.NewMemStore()
		s.AddWebhook(3, 11)
		return storetest.Fixture{Store: s, Member: 7, Platform: 3, Webhook: 11}
	})
}
