package environment

import (
	"strings"
	"testing"
)

func TestCheckDatabase(t *testing.T) {
	cases := []struct {
		name  string
		env   Environment
		db    string
		host  string
		stamp Environment
		want  string // "" means allowed
	}{
		{"live remote", Live, "gateway", "db.internal", "", ""},
		{"live stamped live", Live, "gateway", "db.internal", Live, ""},
		{"live trailing space is still the test database", Live, "payminto_test ", "db.internal", "", "is the test database"},
		{"live case variant of test suffix", Live, "Gateway_Test", "db.internal", "", "ends in _test"},
		{"live stamped test", Live, "gateway", "db.internal", Test, "stamped test"},
		{"test named", Test, "gateway_test", "db.internal", "", ""},
		{"test named stamped test", Test, "gateway_test", "db.internal", Test, ""},
		{"test named stamped live", Test, "gateway_test", "db.internal", Live, "stamped live"},
		{"test remote unnamed", Test, "payminto", "db.internal", "", "neither named *_test nor local"},
		{"test loopback new", Test, "payminto", "127.0.0.1", "", ""},
		{"test loopback stamped test", Test, "payminto", "localhost", Test, ""},
		{"test loopback stamped live", Test, "payminto", "localhost", Live, "stamped live"},
		{"unknown stamp", Test, "gateway_test", "db.internal", "prod", "unknown environment stamp"},
		{"empty name", Live, "  ", "db.internal", "", "database name is empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckDatabase(tc.env, DatabasePolicy{Name: tc.db, TestName: "payminto_test", Host: tc.host, Stamp: tc.stamp})
			if tc.want == "" {
				if err != nil {
					t.Fatalf("CheckDatabase = %v, want allowed", err)
				}
				return
			}
			if !IsBootRefusal(err) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("CheckDatabase = %v, want refusal mentioning %q", err, tc.want)
			}
		})
	}
}

func TestCheckDatabase_AllowNameAdmitsOneRemoteTestDatabase(t *testing.T) {
	allowed := DatabasePolicy{Name: "payminto_staging", TestName: "payminto_test", AllowName: "Payminto_Staging", Host: "db.internal"}
	if err := CheckDatabase(Test, allowed); err != nil {
		t.Fatalf("allowed remote name refused: %v", err)
	}
	other := allowed
	other.Name = "payminto_other"
	if err := CheckDatabase(Test, other); !IsBootRefusal(err) || !strings.Contains(err.Error(), "GATEWAY_TEST_DATABASE_NAME") {
		t.Fatalf("other remote name = %v, want refusal naming GATEWAY_TEST_DATABASE_NAME", err)
	}
	stampedLive := allowed
	stampedLive.Stamp = Live
	if err := CheckDatabase(Test, stampedLive); !IsBootRefusal(err) {
		t.Fatalf("allowed name stamped live accepted: %v", err)
	}
	if err := CheckDatabase(Live, allowed); err != nil {
		t.Fatalf("allow name must not affect live: %v", err)
	}
}
