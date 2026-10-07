package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestHelperBoot is the subprocess entry: it runs main with the environment the parent test sets.
func TestHelperBoot(t *testing.T) {
	if os.Getenv("GATEWAY_BOOT_HELPER") != "1" {
		t.Skip("helper process only")
	}
	main()
}

// bootProcess runs main in a child process with exactly env and returns its exit code and output.
func bootProcess(t *testing.T, env map[string]string) (*exec.Cmd, func() (int, string)) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperBoot$", "-test.v")
	cmd.Env = []string{"GATEWAY_BOOT_HELPER=1", "PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME")}
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var out strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &out
	wait := func() (int, string) {
		err := cmd.Wait()
		if err == nil {
			return 0, out.String()
		}
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode(), out.String()
		}
		return -1, out.String() + "\n" + err.Error()
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	return cmd, wait
}

// liveEnv is a live configuration that passes config.Load; tests mutate one thing at a time.
func liveEnv() map[string]string {
	return map[string]string{
		"GATEWAY_ENVIRONMENT":     "live",
		"SERVER":                  "production",
		"POSTGRES_HOST":           "db.internal",
		"POSTGRES_DATABASE":       "gateway",
		"POSTGRES_PASSWORD":       "a-strong-database-secret-value",
		"POSTGRES_SSL_MODE":       "verify-full",
		"JWT_SECRET":              "a-strong-jwt-secret-value-with-32-plus-chars",
		"BLOCKCHAIN_NETWORK_TYPE": "mainnet",
	}
}

func requireRefusal(t *testing.T, env map[string]string, want string) {
	t.Helper()
	_, wait := bootProcess(t, env)
	code, out := wait()
	if code == 0 {
		t.Fatalf("process started; want a non-zero exit\n%s", out)
	}
	if !strings.Contains(out, want) {
		t.Fatalf("exit %d but the message does not say %q:\n%s", code, want, out)
	}
	if strings.Contains(out, "database:") && !strings.Contains(out, "environment:") {
		t.Fatalf("refusal came from the database, not the environment gate:\n%s", out)
	}
}

func TestBoot_LiveWithDevKeystoreExitsNonZero(t *testing.T) {
	env := liveEnv()
	env["DEV_KEYSTORE"] = "true"
	requireRefusal(t, env, "development keystore")
}

func TestBoot_LiveWithLocalVaultMasterKeyExitsNonZero(t *testing.T) {
	env := liveEnv()
	env["AES_KEY"] = "local-master-key-material"
	requireRefusal(t, env, "local vault master key")
}

func TestBoot_LiveWithMockProviderExitsNonZero(t *testing.T) {
	env := liveEnv()
	env["CUSTODY_PROVIDER"] = "mock"
	requireRefusal(t, env, "slot custody resolved to the mock provider in live")
}

func TestBoot_LiveAgainstTestDatabaseExitsNonZero(t *testing.T) {
	env := liveEnv()
	env["POSTGRES_DATABASE"] = "gateway_test"
	requireRefusal(t, env, "ends in _test")
}

func TestBoot_TestAgainstRemoteNonTestDatabaseRefuses(t *testing.T) {
	requireRefusal(t, map[string]string{
		"GATEWAY_ENVIRONMENT": "test",
		"POSTGRES_HOST":       "db.internal",
		"POSTGRES_DATABASE":   "payminto",
	}, "neither named *_test nor local")
}

func TestBoot_UnknownEnvironmentRefuses(t *testing.T) {
	requireRefusal(t, map[string]string{"GATEWAY_ENVIRONMENT": "sandbox"}, "GATEWAY_ENVIRONMENT")
}

func TestBoot_LiveWithDevelopmentJWTSecretExitsNonZero(t *testing.T) {
	env := liveEnv()
	env["JWT_SECRET"] = "payminto-development-jwt-secret-not-for-production"
	requireRefusal(t, env, "JWT_SECRET")
}
