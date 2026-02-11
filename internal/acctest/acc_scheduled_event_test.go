//go:build acctest
// +build acctest

package acctest

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccScheduledEvent_VoiceBasic(t *testing.T) {
	token := testAccToken(t)
	guildID := testAccGuildID(t)

	suffix := acctest.RandStringFromCharSet(8, "abcdefghijklmnopqrstuvwxyz0123456789")
	voiceName := fmt.Sprintf("tf-acc-voice-%s", strings.ToLower(suffix))
	eventName := fmt.Sprintf("tf-acc-event-%s", strings.ToLower(suffix))
	start := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)

	cfg1 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_channel" "voice" {
  server_id = %q
  type      = "voice"
  name      = %q
}

resource "discord_scheduled_event" "ev" {
  server_id             = %q
  name                  = %q
  entity_type           = 2
  channel_id            = discord_channel.voice.id
  scheduled_start_time  = %q
  description           = "created by terraform acc test"
}
`, token, guildID, voiceName, guildID, eventName, start)

	cfg2 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_channel" "voice" {
  server_id = %q
  type      = "voice"
  name      = %q
}

resource "discord_scheduled_event" "ev" {
  server_id             = %q
  name                  = %q
  entity_type           = 2
  channel_id            = discord_channel.voice.id
  scheduled_start_time  = %q
  description           = "updated by terraform acc test"
}
`, token, guildID, voiceName, guildID, eventName, start)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_scheduled_event.ev", "name", eventName),
					resource.TestCheckResourceAttr("discord_scheduled_event.ev", "entity_type", "2"),
				),
			},
			{
				Config: cfg2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_scheduled_event.ev", "description", "updated by terraform acc test"),
				),
			},
		},
	})
}
