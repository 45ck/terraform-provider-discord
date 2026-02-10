//go:build acctest
// +build acctest

package acctest

import (
	"os"
	"testing"
)

func TestAccEnvSanity(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	if os.Getenv("DISCORD_TOKEN") == "" {
		t.Fatal("DISCORD_TOKEN must be set for acceptance tests")
	}
	if os.Getenv("DISCORD_GUILD_ID") == "" && os.Getenv("DISCORD_SERVER_ID") == "" {
		t.Fatal("DISCORD_GUILD_ID (or legacy DISCORD_SERVER_ID) must be set for acceptance tests")
	}
}
