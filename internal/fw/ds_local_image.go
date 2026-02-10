package fw

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/polds/imgbase64"
)

func NewLocalImageDataSource() datasource.DataSource {
	return &localImageDataSource{}
}

type localImageDataSource struct{}

type localImageModel struct {
	ID      types.String `tfsdk:"id"`
	File    types.String `tfsdk:"file"`
	DataURI types.String `tfsdk:"data_uri"`
}

func (d *localImageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_local_image"
}

func (d *localImageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"file": schema.StringAttribute{
				Required: true,
			},
			"data_uri": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *localImageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data localImageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	img, err := imgbase64.FromLocal(data.File.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to process image", fmt.Sprintf("%s: %s", data.File.ValueString(), err.Error()))
		return
	}

	data.DataURI = types.StringValue(img)
	data.ID = types.StringValue(strconv.Itoa(discord.Hashcode(img)))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
