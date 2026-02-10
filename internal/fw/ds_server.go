package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewServerDataSource() datasource.DataSource {
	return &serverDataSource{}
}

type serverDataSource struct {
	c *discord.RestClient
}

type serverModel struct {
	ID                          types.String `tfsdk:"id"`
	ServerID                    types.String `tfsdk:"server_id"`
	Name                        types.String `tfsdk:"name"`
	Region                      types.String `tfsdk:"region"`
	DefaultMessageNotifications types.Int64  `tfsdk:"default_message_notifications"`
	VerificationLevel           types.Int64  `tfsdk:"verification_level"`
	ExplicitContentFilter       types.Int64  `tfsdk:"explicit_content_filter"`
	AfkTimeout                  types.Int64  `tfsdk:"afk_timeout"`
	IconHash                    types.String `tfsdk:"icon_hash"`
	SplashHash                  types.String `tfsdk:"splash_hash"`
	AfkChannelID                types.String `tfsdk:"afk_channel_id"`
	OwnerID                     types.String `tfsdk:"owner_id"`
}

func (d *serverDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (d *serverDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":                            schema.StringAttribute{Computed: true},
			"server_id":                     schema.StringAttribute{Optional: true},
			"name":                          schema.StringAttribute{Optional: true, Description: "Lookup by name is not supported for bot tokens; use server_id."},
			"region":                        schema.StringAttribute{Computed: true},
			"default_message_notifications": schema.Int64Attribute{Computed: true},
			"verification_level":            schema.Int64Attribute{Computed: true},
			"explicit_content_filter":       schema.Int64Attribute{Computed: true},
			"afk_timeout":                   schema.Int64Attribute{Computed: true},
			"icon_hash":                     schema.StringAttribute{Computed: true},
			"splash_hash":                   schema.StringAttribute{Computed: true},
			"afk_channel_id":                schema.StringAttribute{Computed: true},
			"owner_id":                      schema.StringAttribute{Computed: true},
		},
	}
}

func (d *serverDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *serverDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data serverModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Name.IsNull() && data.Name.ValueString() != "" {
		resp.Diagnostics.AddError("Unsupported lookup", "discord_server data source does not support lookup by name for bot tokens; set server_id")
		return
	}

	serverID := data.ServerID.ValueString()
	if serverID == "" {
		resp.Diagnostics.AddError("Missing server_id", "either server_id or name must be set")
		return
	}

	var guild discordRestGuild
	if err := d.c.DoJSON(ctx, "GET", "/guilds/"+serverID, nil, nil, &guild); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	data.ID = types.StringValue(guild.ID)
	data.ServerID = types.StringValue(guild.ID)
	data.Name = types.StringValue(guild.Name)
	data.Region = types.StringValue(guild.Region)
	data.DefaultMessageNotifications = types.Int64Value(int64(guild.DefaultMessageNotifications))
	data.VerificationLevel = types.Int64Value(int64(guild.VerificationLevel))
	data.ExplicitContentFilter = types.Int64Value(int64(guild.ExplicitContentFilter))
	data.AfkTimeout = types.Int64Value(int64(guild.AfkTimeout))
	data.IconHash = types.StringValue(guild.Icon)
	data.SplashHash = types.StringValue(guild.Splash)
	if guild.AfkChannelID != "" {
		data.AfkChannelID = types.StringValue(guild.AfkChannelID)
	} else {
		data.AfkChannelID = types.StringNull()
	}
	if guild.OwnerID != "" {
		data.OwnerID = types.StringValue(guild.OwnerID)
	} else {
		data.OwnerID = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// discordRestGuild matches fields we need from GET /guilds/{id}.
type discordRestGuild struct {
	ID                          string `json:"id"`
	Name                        string `json:"name"`
	Region                      string `json:"region"`
	DefaultMessageNotifications int    `json:"default_message_notifications"`
	VerificationLevel           int    `json:"verification_level"`
	ExplicitContentFilter       int    `json:"explicit_content_filter"`
	AfkTimeout                  int    `json:"afk_timeout"`
	AfkChannelID                string `json:"afk_channel_id"`
	OwnerID                     string `json:"owner_id"`
	Icon                        string `json:"icon"`
	Splash                      string `json:"splash"`
	SystemChannelID             string `json:"system_channel_id"`
}

func (g discordRestGuild) String() string {
	return fmt.Sprintf("guild(%s)", g.ID)
}
