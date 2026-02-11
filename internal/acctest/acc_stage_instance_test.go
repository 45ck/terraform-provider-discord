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

func TestAccStageInstance_Basic(t *testing.T) {
	if os.Getenv("DISCORD_ENABLE_STAGE_INSTANCE_TEST") == "" {
		t.Skip("DISCORD_ENABLE_STAGE_INSTANCE_TEST not set")
	}

	token := testAccToken(t)
	guildID := testAccGuildID(t)

	suffix := acctest.RandStringFromCharSet(8, "abcdefghijklmnopqrstuvwxyz0123456789")
	stageName := fmt.Sprintf("tf-acc-stage-%s", strings.ToLower(suffix))

	cfg1 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_channel" "stage" {
  server_id = %q
  type      = "stage"
  name      = %q
}

resource "discord_stage_instance" "si" {
  channel_id = discord_channel.stage.id
  topic      = "hello from acc test"
}
`, token, guildID, stageName)

	cfg2 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_channel" "stage" {
  server_id = %q
  type      = "stage"
  name      = %q
}

resource "discord_stage_instance" "si" {
  channel_id = discord_channel.stage.id
  topic      = "hello from acc test (edited)"
}
`, token, guildID, stageName)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_stage_instance.si", "topic", "hello from acc test"),
				),
			},
			{
				Config: cfg2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_stage_instance.si", "topic", "hello from acc test (edited)"),
				),
			},
		},
	})
}
