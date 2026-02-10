//go:build acctest
// +build acctest

package discord

import (
	"os"
	"testing"
)

// Acceptance tests are intentionally minimal and opt-in.
// Run with:
//
//	TF_ACC=1 DISCORD_TOKEN=... DISCORD_SERVER_ID=... go test ./discord -run TestAcc -v
func TestAccEnvSanity(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	if os.Getenv("DISCORD_TOKEN") == "" {
		t.Fatal("DISCORD_TOKEN must be set for acceptance tests")
	}
	if os.Getenv("DISCORD_SERVER_ID") == "" {
		t.Fatal("DISCORD_SERVER_ID must be set for acceptance tests")
	}
}
