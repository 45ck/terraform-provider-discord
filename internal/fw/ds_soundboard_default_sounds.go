package fw

import (
	"context"
	"fmt"

	"github.com/45ck/terraform-provider-discord/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewSoundboardDefaultSoundsDataSource() datasource.DataSource {
	return &soundboardDefaultSoundsDataSource{}
}

type soundboardDefaultSoundsDataSource struct {
	c *discord.RestClient
}

type soundboardDefaultSoundsModel struct {
	ID    types.String           `tfsdk:"id"`
	Sound []soundboardSoundModel `tfsdk:"sound"`
}

func (d *soundboardDefaultSoundsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_soundboard_default_sounds"
}

func (d *soundboardDefaultSoundsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
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

func (d *soundboardDefaultSoundsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *soundboardDefaultSoundsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data soundboardDefaultSoundsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var out []restSoundboardSound
	if err := d.c.DoJSON(ctx, "GET", "/soundboard-default-sounds", nil, nil, &out); err != nil {
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

	data.ID = types.StringValue(fmt.Sprintf("%d", len(sounds)))
	data.Sound = sounds
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
