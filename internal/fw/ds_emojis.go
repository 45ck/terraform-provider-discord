package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewEmojisDataSource() datasource.DataSource {
	return &emojisDataSource{}
}

type emojisDataSource struct {
	c *discord.RestClient
}

type restEmojiLite struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Managed  bool   `json:"managed"`
	Animated bool   `json:"animated"`
}

type emojiModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Managed  types.Bool   `tfsdk:"managed"`
	Animated types.Bool   `tfsdk:"animated"`
}

type emojisModel struct {
	ID       types.String `tfsdk:"id"`
	ServerID types.String `tfsdk:"server_id"`
	Emoji    []emojiModel `tfsdk:"emoji"`
}

func (d *emojisDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_emojis"
}

func (d *emojisDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{Required: true},
			"emoji": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":       schema.StringAttribute{Computed: true},
						"name":     schema.StringAttribute{Computed: true},
						"managed":  schema.BoolAttribute{Computed: true},
						"animated": schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *emojisDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *emojisDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data emojisModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := data.ServerID.ValueString()
	var out []restEmojiLite
	if err := d.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/emojis", serverID), nil, nil, &out); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	emojis := make([]emojiModel, 0, len(out))
	for _, e := range out {
		emojis = append(emojis, emojiModel{
			ID:       types.StringValue(e.ID),
			Name:     types.StringValue(e.Name),
			Managed:  types.BoolValue(e.Managed),
			Animated: types.BoolValue(e.Animated),
		})
	}

	data.ID = types.StringValue(serverID)
	data.Emoji = emojis
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
