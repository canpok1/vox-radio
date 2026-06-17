package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile_DefaultAbsent(t *testing.T) {
	chdirTemp(t)
	if err := loadEnvFile(".env", false); err != nil {
		t.Errorf("default .env absent: expected no error, got %v", err)
	}
}

func TestLoadEnvFile_DefaultPresent(t *testing.T) {
	const key = "VOXRADIO_TEST_LOADENV_DEFAULT"
	dir := chdirTemp(t)
	restoreEnv(t, key)

	content := key + "=loaded_from_file\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadEnvFile(".env", false); err != nil {
		t.Fatalf("default .env present: unexpected error: %v", err)
	}
	if got := os.Getenv(key); got != "loaded_from_file" {
		t.Errorf("expected %s=loaded_from_file, got %q", key, got)
	}
}

func TestLoadEnvFile_ExplicitAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.env")
	if err := loadEnvFile(path, true); err == nil {
		t.Error("explicit absent: expected error, got nil")
	}
}

func TestLoadEnvFile_ExplicitPresent(t *testing.T) {
	const key = "VOXRADIO_TEST_LOADENV_EXPLICIT"
	dir := t.TempDir()
	restoreEnv(t, key)

	path := filepath.Join(dir, "custom.env")
	content := key + "=explicit_value\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadEnvFile(path, true); err != nil {
		t.Fatalf("explicit present: unexpected error: %v", err)
	}
	if got := os.Getenv(key); got != "explicit_value" {
		t.Errorf("expected %s=explicit_value, got %q", key, got)
	}
}

func TestLoadEnvFile_OSEnvPriority(t *testing.T) {
	const key = "VOXRADIO_TEST_LOADENV_PRIORITY"
	dir := chdirTemp(t)
	t.Setenv(key, "from_os")

	content := key + "=from_file\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadEnvFile(".env", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := os.Getenv(key); got != "from_os" {
		t.Errorf("OS env var should take priority: expected from_os, got %q", got)
	}
}

func TestLoadEnvFile_ParseError(t *testing.T) {
	dir := chdirTemp(t)
	// unterminated quoted value causes a parse error in godotenv
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("KEY=\"unclosed\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := loadEnvFile(".env", false); err == nil {
		t.Error("parse error: expected error for malformed .env, got nil")
	}
}

// restoreEnv unsets key and restores its original value on cleanup.
func restoreEnv(t *testing.T, key string) {
	t.Helper()
	orig, wasSet := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv(key, orig)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}
