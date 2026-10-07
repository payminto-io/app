package database

import (
	"strings"
	"testing"

	"github.com/payminto/payminto/backend/internal/config"
)

func TestVerifiedDSN_RoundTripsAwkwardPasswords(t *testing.T) {
	for _, password := range []string{`plain`, `ends-with-backslash\\`, `it's`, `a\\'b`, `x\\' dbname=payminto_test options=-csearch_path=test application_name=`} {
		cfg := config.DatabaseConfig{Host: "db.internal", Port: 5432, Username: "u", Password: password, Database: "gateway", SSLMode: "verify-full"}
		if _, err := VerifiedDSN(cfg); err != nil {
			t.Errorf("password %q: %v", password, err)
		}
	}
}

func TestVerifiedDSN_EscapedNameParsesBackIntact(t *testing.T) {
	// config.Load rejects such a name; for any other caller the quoting keeps it one value.
	cfg := config.DatabaseConfig{Host: "db.internal", Port: 5432, Username: "u", Password: "p", Database: "gateway' dbname='payminto_test", SSLMode: "verify-full"}
	dsn, err := VerifiedDSN(cfg)
	if err != nil {
		t.Fatalf("VerifiedDSN = %v", err)
	}
	if !strings.Contains(dsn, `dbname='gateway\' dbname=\'payminto_test'`) {
		t.Fatalf("DSN = %s", dsn)
	}
}
