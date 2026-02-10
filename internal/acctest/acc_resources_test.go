//go:build acctest
// +build acctest

package acctest

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aequasi/discord-terraform/internal/fw"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	ptacctest "github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"discord": providerserver.NewProtocol6WithError(fw.New("acctest")()),
	}
}

func TestAccChannelAndMessage_Basic(t *testing.T) {
	token := testAccToken(t)
	guildID := testAccGuildID(t)

	suffix := ptacctest.RandStringFromCharSet(8, "abcdefghijklmnopqrstuvwxyz0123456789")
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
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
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
