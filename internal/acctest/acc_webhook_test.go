//go:build acctest
// +build acctest

package acctest

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhook_Basic(t *testing.T) {
	token := testAccToken(t)
	guildID := testAccGuildID(t)

	suffix := acctest.RandStringFromCharSet(8, "abcdefghijklmnopqrstuvwxyz0123456789")
	channelName := fmt.Sprintf("tf-acc-webhook-%s", strings.ToLower(suffix))
	webhookName1 := fmt.Sprintf("tf-acc-webhook-%s", strings.ToLower(suffix))
	webhookName2 := fmt.Sprintf("tf-acc-webhook2-%s", strings.ToLower(suffix))

	cfg1 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_channel" "ch" {
  server_id = %q
  type      = "text"
  name      = %q
}

resource "discord_webhook" "wh" {
  channel_id = discord_channel.ch.id
  name       = %q
}
`, token, guildID, channelName, webhookName1)

	cfg2 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_channel" "ch" {
  server_id = %q
  type      = "text"
  name      = %q
}

resource "discord_webhook" "wh" {
  channel_id = discord_channel.ch.id
  name       = %q
}
`, token, guildID, channelName, webhookName2)

	// Keep this lightweight: in some Discord responses the token/url may be omitted.
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_webhook.wh", "name", webhookName1),
				),
			},
			{
				Config: cfg2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_webhook.wh", "name", webhookName2),
				),
			},
		},
	})
}

func TestAccSticker_Basic(t *testing.T) {
	token := testAccToken(t)
	guildID := testAccGuildID(t)

	filePath := os.Getenv("DISCORD_STICKER_FILE_PATH")
	if filePath == "" {
		t.Skip("DISCORD_STICKER_FILE_PATH not set")
	}

	suffix := acctest.RandStringFromCharSet(8, "abcdefghijklmnopqrstuvwxyz0123456789")
	name := fmt.Sprintf("tf-acc-sticker-%s", strings.ToLower(suffix))

	cfg := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_sticker" "s" {
  server_id    = %q
  name         = %q
  description  = "managed by terraform acceptance tests"
  tags         = "🙂"
  file_path    = %q
}
`, token, guildID, name, filePath)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_sticker.s", "name", name),
				),
			},
		},
	})
}
