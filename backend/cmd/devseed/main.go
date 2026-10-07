// Command devseed provisions disposable local-only admin and merchant logins.
// It refuses non-development and non-loopback database targets. Plaintext
// credentials are written once to a caller-controlled mode-0600 file.
package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/payminto/payminto/backend/internal/environment"
	"github.com/payminto/payminto/backend/internal/ledger"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/payminto/payminto/backend/internal/config"
	"github.com/payminto/payminto/backend/internal/database"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/modules"
	"github.com/payminto/payminto/backend/internal/service"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type credential struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	APIKey     string `json:"apiKey"`
	PlatformID uint   `json:"platformId"`
}

type credentialFile struct {
	Warning  string     `json:"warning"`
	Admin    credential `json:"admin"`
	Merchant credential `json:"merchant"`
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.Server.Environment != config.EnvironmentDevelopment || !localHost(cfg.Database.Host) {
		log.Fatal("devseed is restricted to DEVELOPMENT with a loopback database host")
	}
	envModule, err := modules.WireEnvironment(modules.Deps{Config: cfg})
	if err != nil {
		log.Fatalf("environment: %v", err)
	}
	if envModule.Environment != environment.Test {
		log.Fatalf("devseed seeds test money only; GATEWAY_ENVIRONMENT is %s", envModule.Environment)
	}

	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	if err := envModule.VerifyDatabase(ctx, db); err != nil {
		log.Fatalf("environment: %v", err)
	}
	if err := envModule.VerifySchema(ctx, db); err != nil {
		log.Fatalf("environment: %v", err)
	}
	if err := envModule.Stamp(ctx, db); err != nil {
		log.Fatalf("environment: %v", err)
	}

	// RBAC role names come from the catalog seeds, not from member_type.
	const adminRole = "owner"

	adminPassword := randomSecret("Adm-")
	merchantPassword := randomSecret("Mer-")
	adminKey, err := service.GenerateAPIKey()
	if err != nil {
		log.Fatal(err)
	}
	merchantKey, err := service.GenerateAPIKey()
	if err != nil {
		log.Fatal(err)
	}

	var output credentialFile
	err = db.Transaction(func(tx *gorm.DB) error {
		admin, err := upsertMember(tx, "admin@payminto.dev", "Local Admin", "root", adminPassword)
		if err != nil {
			return err
		}
		adminPlatform, err := existingOrNewPlatform(tx, admin.ID, "Default Project")
		if err != nil {
			return err
		}
		// "root" above is the member_type; the RBAC role is a separate namespace and
		// migrations/seeds/*/9005_seed_rbac.sql seeds no role by that name. The widest
		// seeded role is "owner" (every permission), which is what a platform root holds.
		if err := assignRole(tx, admin.ID, adminPlatform.ID, adminRole); err != nil {
			return err
		}
		if err := replaceDevKey(tx, admin.ID, adminPlatform.ID, adminRole, "Local development admin key", adminKey); err != nil {
			return err
		}

		merchant, err := upsertMember(tx, "merchant@payminto.dev", "Demo Merchant", "internal", merchantPassword)
		if err != nil {
			return err
		}
		merchantPlatform, err := existingOrNewPlatform(tx, merchant.ID, "Demo Merchant")
		if err != nil {
			return err
		}
		if err := assignRole(tx, merchant.ID, merchantPlatform.ID, "owner"); err != nil {
			return err
		}
		if err := replaceDevKey(tx, merchant.ID, merchantPlatform.ID, "owner", "Local development merchant key", merchantKey); err != nil {
			return err
		}

		output = credentialFile{
			Warning:  "LOCAL TESTNET ONLY. Rotate by rerunning devseed. Never deploy or share these credentials.",
			Admin:    credential{Email: "admin@payminto.dev", Password: adminPassword, APIKey: adminKey, PlatformID: adminPlatform.ID},
			Merchant: credential{Email: "merchant@payminto.dev", Password: merchantPassword, APIKey: merchantKey, PlatformID: merchantPlatform.ID},
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	feesModule, err := modules.WireFees(modules.Deps{DB: db, Config: cfg, Ledger: ledger.New(db, ledger.WithEnvironment(envModule.Environment), ledger.WithGuard(envModule.Guard)), LedgerAsset: service.LedgerAssetResolver()})
	if err != nil {
		log.Fatalf("fees: %v", err)
	}
	seeded, err := modules.SeedDevelopmentFeeRules(ctx, envModule.Environment, feesModule.Port, modules.DevFeeMethods(modules.DefaultConnectors(envModule.Environment, cfg)))
	if err != nil {
		log.Fatalf("fee seed: %v", err)
	}
	for _, r := range seeded {
		fmt.Printf("Seeded zero-fee development rule %d for %s/%s (test environment only)\n", r.ID, r.Method, r.Currency)
	}

	path := os.Getenv("DEV_CREDENTIALS_PATH")
	if path == "" {
		path = filepath.Join("..", ".dev-credentials.local.json")
	}
	payload, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o600); err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		log.Fatal(err)
	}
	abs, _ := filepath.Abs(path)
	fmt.Printf("Local testnet credentials written to %s (mode 0600)\n", abs)
}

func upsertMember(tx *gorm.DB, email, name, memberType, password string) (*models.Member, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	var member models.Member
	result := tx.Where("email = ?", email).First(&member)
	if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return nil, result.Error
	}
	passwordHash := string(hash)
	if result.Error == gorm.ErrRecordNotFound {
		member = models.Member{Name: name, Email: &email, Password: &passwordHash, MemberType: memberType, State: "active"}
		if err := tx.Create(&member).Error; err != nil {
			return nil, err
		}
		return &member, nil
	}
	if err := tx.Model(&member).Updates(map[string]any{"name": name, "password": passwordHash, "member_type": memberType, "state": "active", "reset_password_required": false}).Error; err != nil {
		return nil, err
	}
	return &member, nil
}

func existingOrNewPlatform(tx *gorm.DB, memberID uint, name string) (*models.ExternalPlatform, error) {
	var platform models.ExternalPlatform
	err := tx.Table("external_platforms ep").
		Joins("JOIN member_external_platform_roles mepr ON mepr.external_platform_id = ep.id").
		Where("mepr.member_id = ?", memberID).
		Order("ep.id").
		First(&platform).Error
	if err == nil {
		return &platform, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	platform = models.ExternalPlatform{Name: name, Website: "http://localhost:3003", SuccessEndpoint: ""}
	if err := tx.Create(&platform).Error; err != nil {
		return nil, err
	}
	return &platform, nil
}

func assignRole(tx *gorm.DB, memberID, platformID uint, roleName string) error {
	var role models.Role
	if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
		return fmt.Errorf("role %q is not seeded: %w", roleName, err)
	}
	assignment := models.MemberExternalPlatformRole{MemberID: memberID, ExternalPlatformID: platformID, RoleID: role.ID}
	return tx.Where(assignment).FirstOrCreate(&assignment).Error
}

func replaceDevKey(tx *gorm.DB, memberID, platformID uint, roleName, description, raw string) error {
	if err := tx.Model(&models.APIKey{}).Where("external_platform_id = ? AND description = ?", platformID, description).Update("status", "inactive").Error; err != nil {
		return err
	}
	var role models.Role
	if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
		return err
	}
	key := service.NewAPIKeyRow(raw, environment.Test, platformID)
	key.MemberID, key.RoleID, key.Description = &memberID, &role.ID, &description
	return tx.Create(key).Error
}

func randomSecret(prefix string) string {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatal(err)
	}
	return prefix + strings.TrimRight(base64.RawURLEncoding.EncodeToString(bytes), "=")
}

func localHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}
