package discord

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:      schema.TypeString,
				Required:  true,
				Sensitive: true,
			},
			"client_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"secret": {
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"discord_server":              resourceDiscordServer(),
			"discord_category_channel":    resourceDiscordCategoryChannel(),
			"discord_text_channel":        resourceDiscordTextChannel(),
			"discord_voice_channel":       resourceDiscordVoiceChannel(),
			"discord_channel":             resourceDiscordChannel(),
			"discord_channel_order":       resourceDiscordChannelOrder(),
			"discord_channel_permission":  resourceDiscordChannelPermission(),
			"discord_channel_permissions": resourceDiscordChannelPermissions(),
			"discord_invite":              resourceDiscordInvite(),
			"discord_role":                resourceDiscordRole(),
			"discord_role_everyone":       resourceDiscordRoleEveryone(),
			"discord_role_order":          resourceDiscordRoleOrder(),
			"discord_member_roles":        resourceDiscordMemberRoles(),
			"discord_message":             resourceDiscordMessage(),
			"discord_thread":              resourceDiscordThread(),
			"discord_thread_member":       resourceDiscordThreadMember(),
			"discord_system_channel":      resourceDiscordSystemChannel(),
			"discord_welcome_screen":      resourceDiscordWelcomeScreen(),
			"discord_onboarding":          resourceDiscordOnboarding(),
			"discord_automod_rule":        resourceDiscordAutoModRule(),
			"discord_scheduled_event":     resourceDiscordScheduledEvent(),
			"discord_emoji":               resourceDiscordEmoji(),
			"discord_sticker":             resourceDiscordSticker(),
			"discord_webhook":             resourceDiscordWebhook(),
			"discord_stage_instance":      resourceDiscordStageInstance(),
			"discord_member_verification": resourceDiscordMemberVerification(),
			"discord_guild_settings":      resourceDiscordGuildSettings(),
			"discord_ban":                 resourceDiscordBan(),
			"discord_member_timeout":      resourceDiscordMemberTimeout(),
			"discord_member_nickname":     resourceDiscordMemberNickname(),
			"discord_api_resource":        resourceDiscordAPIResource(),
			"discord_soundboard_sound":    resourceDiscordSoundboardSound(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			// Data sources are served by the Framework-side provider via terraform-plugin-mux.
		},

		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	config := Config{
		Token: d.Get("token").(string),
	}

	client, err := config.Client()
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return client, diags
}
