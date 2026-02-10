package fw

import (
	"context"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewSystemChannelDataSource() datasource.DataSource {
	return &systemChannelDataSource{}
}

type systemChannelDataSource struct {
	c *discord.RestClient
}

type systemChannelModel struct {
	ID              types.String `tfsdk:"id"`
	ServerID        types.String `tfsdk:"server_id"`
	SystemChannelID types.String `tfsdk:"system_channel_id"`
}

func (d *systemChannelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_channel"
}

func (d *systemChannelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Computed: true},
			"server_id":         schema.StringAttribute{Required: true},
			"system_channel_id": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *systemChannelDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *systemChannelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data systemChannelModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := data.ServerID.ValueString()
	var guild discordRestGuild
	if err := d.c.DoJSON(ctx, "GET", "/guilds/"+serverID, nil, nil, &guild); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	data.ID = types.StringValue(guild.ID)
	if guild.SystemChannelID == "" {
		data.SystemChannelID = types.StringValue("")
	} else {
		data.SystemChannelID = types.StringValue(guild.SystemChannelID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
