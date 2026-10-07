package api

import (
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/modules"
	"strings"
	"testing"

	"github.com/payminto/payminto/backend/internal/service"
)

// Sweeps are not scoped to a platform, so instance-wide sweep stats must never be on the merchant API.
func TestRouter_NoInstanceWideSweepStatsOnMerchantAPI(t *testing.T) {
	r := NewRouter(RouterConfig{
		Environment:  &modules.EnvironmentModule{Environment: environment.Test},
		Host:         "http://localhost:8080",
		AnalyticsSvc: service.NewAnalyticsService(nil),
		MEPRoleSvc:   &service.MemberExternalPlatformRoleService{},
	})
	var sawSummary bool
	for _, rt := range r.Routes() {
		if strings.HasSuffix(rt.Path, "/analytics/sweeps") {
			t.Fatalf("merchant route %s %s exposes instance-wide sweep stats", rt.Method, rt.Path)
		}
		if strings.HasSuffix(rt.Path, "/analytics/summary") {
			sawSummary = true
		}
	}
	if !sawSummary {
		t.Fatal("analytics routes not registered; test config is wrong")
	}
}
