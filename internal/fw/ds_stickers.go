package fw

import (
	"context"
	"fmt"

	"github.com/45ck/terraform-provider-discord/discord"
	"github.com/45ck/terraform-provider-discord/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewStickersDataSource() datasource.DataSource {
	return &stickersDataSource{}
}

type stickersDataSource struct {
	c *discord.RestClient
}

type restStickerLite struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	FormatType  int    `json:"format_type"`
}

type stickerModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Tags        types.String `tfsdk:"tags"`
	FormatType  types.Int64  `tfsdk:"format_type"`
}

type stickersModel struct {
	ID       types.String   `tfsdk:"id"`
	ServerID types.String   `tfsdk:"server_id"`
	Sticker  []stickerModel `tfsdk:"sticker"`
}

func (d *stickersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stickers"
}

func (d *stickersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"sticker": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"description": schema.StringAttribute{Computed: true},
						"tags":        schema.StringAttribute{Computed: true},
						"format_type": schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *stickersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *stickersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data stickersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := data.ServerID.ValueString()
	var out []restStickerLite
	if err := d.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/stickers", serverID), nil, nil, &out); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	stickers := make([]stickerModel, 0, len(out))
	for _, s := range out {
		stickers = append(stickers, stickerModel{
			ID:          types.StringValue(s.ID),
			Name:        types.StringValue(s.Name),
			Description: types.StringValue(s.Description),
			Tags:        types.StringValue(s.Tags),
			FormatType:  types.Int64Value(int64(s.FormatType)),
		})
	}

	data.ID = types.StringValue(serverID)
	data.Sticker = stickers
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
