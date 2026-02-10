package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewSoundboardSoundsDataSource() datasource.DataSource {
	return &soundboardSoundsDataSource{}
}

type soundboardSoundsDataSource struct {
	c *discord.RestClient
}

type restSoundboardSound struct {
	SoundID   string  `json:"sound_id"`
	Name      string  `json:"name"`
	Volume    float64 `json:"volume"`
	EmojiID   string  `json:"emoji_id"`
	EmojiName string  `json:"emoji_name"`
	Available bool    `json:"available"`
}

type soundboardSoundModel struct {
	SoundID   types.String  `tfsdk:"sound_id"`
	Name      types.String  `tfsdk:"name"`
	Volume    types.Float64 `tfsdk:"volume"`
	EmojiID   types.String  `tfsdk:"emoji_id"`
	EmojiName types.String  `tfsdk:"emoji_name"`
	Available types.Bool    `tfsdk:"available"`
}

type soundboardSoundsModel struct {
	ID       types.String           `tfsdk:"id"`
	ServerID types.String           `tfsdk:"server_id"`
	Sound    []soundboardSoundModel `tfsdk:"sound"`
}

func (d *soundboardSoundsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_soundboard_sounds"
}

func (d *soundboardSoundsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{Required: true},
			"sound": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"sound_id":   schema.StringAttribute{Computed: true},
						"name":       schema.StringAttribute{Computed: true},
						"volume":     schema.Float64Attribute{Computed: true},
						"emoji_id":   schema.StringAttribute{Computed: true},
						"emoji_name": schema.StringAttribute{Computed: true},
						"available":  schema.BoolAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *soundboardSoundsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *soundboardSoundsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data soundboardSoundsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := data.ServerID.ValueString()
	var out []restSoundboardSound
	if err := d.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/soundboard-sounds", serverID), nil, nil, &out); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	sounds := make([]soundboardSoundModel, 0, len(out))
	for _, s := range out {
		sounds = append(sounds, soundboardSoundModel{
			SoundID:   types.StringValue(s.SoundID),
			Name:      types.StringValue(s.Name),
			Volume:    types.Float64Value(s.Volume),
			EmojiID:   types.StringValue(s.EmojiID),
			EmojiName: types.StringValue(s.EmojiName),
			Available: types.BoolValue(s.Available),
		})
	}

	data.ID = types.StringValue(serverID)
	data.Sound = sounds
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
