package helper

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadClientID(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "config.json")
	noEnv := func(string) string { return "" }

	if got := loadClientID(noEnv, config); got != "" {
		t.Fatalf("missing config: %q", got)
	}
	for content, want := range map[string]string{
		`{"clientId": "123456789012345678"}`: "123456789012345678",
		`{"clientId": 12345678901234567}`:    "12345678901234567",
		`{"clientId": "12345"}`:              "",
		`{"clientId": null}`:                 "",
		`not json`:                           "",
	} {
		os.WriteFile(config, []byte(content), 0o644)
		if got := loadClientID(noEnv, config); got != want {
			t.Errorf("%s: got %q, want %q", content, got, want)
		}
	}
	env := func(k string) string {
		if k == "DISCORD_CLIENT_ID" {
			return "98765432109876543"
		}
		return ""
	}
	if got := loadClientID(env, config); got != "98765432109876543" {
		t.Fatalf("env override: %q", got)
	}
}
