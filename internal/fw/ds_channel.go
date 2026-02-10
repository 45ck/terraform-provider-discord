package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewChannelDataSource() datasource.DataSource {
	return &channelDataSource{}
}

type channelDataSource struct {
	c *discord.RestClient
}

type restChannelLite struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     uint   `json:"type"`
	ParentID string `json:"parent_id"`
}

type channelModel struct {
	ID       types.String `tfsdk:"id"`
	ServerID types.String `tfsdk:"server_id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	ParentID types.String `tfsdk:"parent_id"`
}

func (d *channelDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel"
}

func (d *channelDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{Required: true},
			"name":      schema.StringAttribute{Required: true},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Optional channel type filter (text, voice, category, news, stage, forum, media).",
			},
			"parent_id": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *channelDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *channelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data channelModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := data.ServerID.ValueString()
	name := data.Name.ValueString()
	wantType := data.Type.ValueString()

	var channels []restChannelLite
	if err := d.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/channels", serverID), nil, nil, &channels); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	var matches []restChannelLite
	for _, ch := range channels {
		if ch.Name != name {
			continue
		}
		if wantType != "" {
			got, ok := discord.GetTextChannelType(ch.Type)
			if !ok || got != wantType {
				continue
			}
		}
		matches = append(matches, ch)
	}

	if len(matches) == 0 {
		resp.Diagnostics.AddError("Channel not found", fmt.Sprintf("no channel named %q found in server %s", name, serverID))
		return
	}
	if len(matches) > 1 {
		resp.Diagnostics.AddError("Ambiguous channel", fmt.Sprintf("multiple channels named %q found in server %s; specify a more precise filter", name, serverID))
		return
	}

	ch := matches[0]
	data.ID = types.StringValue(ch.ID)
	data.ParentID = types.StringValue(ch.ParentID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
