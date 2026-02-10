//go:build acctest
// +build acctest

package discord

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func testAccToken(t *testing.T) string {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
	v := os.Getenv("DISCORD_TOKEN")
	if v == "" {
		t.Fatal("DISCORD_TOKEN must be set for acceptance tests")
	}
	return v
}

func testAccGuildID(t *testing.T) string {
	t.Helper()
	v := os.Getenv("DISCORD_GUILD_ID")
	if v == "" {
		v = os.Getenv("DISCORD_SERVER_ID")
	}
	if v == "" {
		t.Fatal("DISCORD_GUILD_ID (or legacy DISCORD_SERVER_ID) must be set for acceptance tests")
	}
	return v
}

func testAccProviderFactories(t *testing.T) map[string]func() (*schema.Provider, error) {
	t.Helper()
	return map[string]func() (*schema.Provider, error){
		"discord": func() (*schema.Provider, error) {
			return Provider(), nil
		},
	}
}

func TestAccChannelAndMessage_Basic(t *testing.T) {
	token := testAccToken(t)
	guildID := testAccGuildID(t)

	suffix := acctest.RandString(8)
	channelName := fmt.Sprintf("tf-acc-%s", strings.ToLower(suffix))

	cfg1 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_channel" "test" {
  server_id = %q
  type      = "text"
  name      = %q
  topic     = "hello from terraform acc"
}

resource "discord_message" "msg" {
  channel_id = discord_channel.test.id
  content    = "hello world"
  pinned     = true
}
`, token, guildID, channelName)

	cfg2 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_channel" "test" {
  server_id = %q
  type      = "text"
  name      = %q
  topic     = "hello from terraform acc"
}

resource "discord_message" "msg" {
  channel_id = discord_channel.test.id
  content    = "hello world (edited)"
  pinned     = true
}
`, token, guildID, channelName)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: cfg1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_channel.test", "name", channelName),
					resource.TestCheckResourceAttr("discord_message.msg", "content", "hello world"),
					resource.TestCheckResourceAttr("discord_message.msg", "pinned", "true"),
				),
			},
			{
				Config: cfg2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_message.msg", "content", "hello world (edited)"),
				),
			},
		},
	})
}

func TestAccRole_Basic(t *testing.T) {
	token := testAccToken(t)
	guildID := testAccGuildID(t)

	suffix := acctest.RandString(8)
	roleName := fmt.Sprintf("tf-acc-role-%s", suffix)

	cfg1 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_role" "test" {
  server_id          = %q
  name               = %q
  permissions_bits64 = "0"
  color              = 0
  hoist              = false
  mentionable        = false
  position           = 1
}
`, token, guildID, roleName)

	cfg2 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_role" "test" {
  server_id          = %q
  name               = %q
  permissions_bits64 = "0"
  color              = 0
  hoist              = false
  mentionable        = true
  position           = 1
}
`, token, guildID, roleName)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories(t),
		Steps: []resource.TestStep{
			{
				Config: cfg1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_role.test", "name", roleName),
					resource.TestCheckResourceAttr("discord_role.test", "mentionable", "false"),
				),
			},
			{
				Config: cfg2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_role.test", "mentionable", "true"),
				),
			},
		},
	})
}
