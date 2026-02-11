//go:build acctest
// +build acctest

package acctest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccThreadAndMember_Basic(t *testing.T) {
	token := testAccToken(t)
	guildID := testAccGuildID(t)

	suffix := acctest.RandStringFromCharSet(8, "abcdefghijklmnopqrstuvwxyz0123456789")
	channelName := fmt.Sprintf("tf-acc-threads-%s", strings.ToLower(suffix))
	threadName := fmt.Sprintf("tf-acc-thread-%s", strings.ToLower(suffix))

	cfg := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_channel" "parent" {
  server_id = %q
  type      = "text"
  name      = %q
}

resource "discord_thread" "t" {
  channel_id = discord_channel.parent.id
  type       = "public_thread"
  name       = %q
}

resource "discord_thread_member" "me" {
  thread_id = discord_thread.t.id
  user_id   = "@me"
}
`, token, guildID, channelName, threadName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_channel.parent", "name", channelName),
					resource.TestCheckResourceAttr("discord_thread.t", "name", threadName),
					resource.TestCheckResourceAttr("discord_thread.t", "type", "public_thread"),
					resource.TestCheckResourceAttr("discord_thread_member.me", "user_id", "@me"),
				),
			},
		},
	})
}
