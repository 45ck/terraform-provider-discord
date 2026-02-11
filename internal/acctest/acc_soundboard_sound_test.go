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

func TestAccSoundboardSound_Basic(t *testing.T) {
	if os.Getenv("DISCORD_ENABLE_SOUNDBOARD_TEST") == "" {
		t.Skip("DISCORD_ENABLE_SOUNDBOARD_TEST not set")
	}

	token := testAccToken(t)
	guildID := testAccGuildID(t)

	filePath := os.Getenv("DISCORD_SOUNDBOARD_FILE_PATH")
	if filePath == "" {
		t.Skip("DISCORD_SOUNDBOARD_FILE_PATH not set")
	}

	suffix := acctest.RandStringFromCharSet(8, "abcdefghijklmnopqrstuvwxyz0123456789")
	name1 := fmt.Sprintf("tf-acc-sound-%s", strings.ToLower(suffix))
	name2 := fmt.Sprintf("tf-acc-sound2-%s", strings.ToLower(suffix))

	cfg1 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_soundboard_sound" "s" {
  server_id        = %q
  name             = %q
  sound_file_path  = %q
  volume           = 1.0
  emoji_name       = "🔊"
}
`, token, guildID, name1, filePath)

	cfg2 := fmt.Sprintf(`
provider "discord" {
  token = %q
}

resource "discord_soundboard_sound" "s" {
  server_id        = %q
  name             = %q
  sound_file_path  = %q
  volume           = 0.75
  emoji_name       = "🔊"
}
`, token, guildID, name2, filePath)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg1,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_soundboard_sound.s", "name", name1),
				),
			},
			{
				Config: cfg2,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("discord_soundboard_sound.s", "name", name2),
				),
			},
		},
	})
}
