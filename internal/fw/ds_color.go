package fw

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/go-playground/colors.v1"
)

func NewColorDataSource() datasource.DataSource {
	return &colorDataSource{}
}

type colorDataSource struct{}

type colorModel struct {
	Hex types.String `tfsdk:"hex"`
	RGB types.String `tfsdk:"rgb"`
	Dec types.Int64  `tfsdk:"dec"`
	ID  types.String `tfsdk:"id"`
}

func (d *colorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_color"
}

func (d *colorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"hex": schema.StringAttribute{
				Optional: true,
			},
			"rgb": schema.StringAttribute{
				Optional: true,
			},
			"dec": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (d *colorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data colorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasHex := !data.Hex.IsNull() && data.Hex.ValueString() != ""
	hasRGB := !data.RGB.IsNull() && data.RGB.ValueString() != ""
	if hasHex == hasRGB { // both true or both false
		resp.Diagnostics.AddError(
			"Invalid configuration",
			"Exactly one of hex or rgb must be set.",
		)
		return
	}

	var hex string
	if hasHex {
		clr, err := colors.ParseHEX(data.Hex.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse hex", err.Error())
			return
		}
		hex = clr.String()
	}
	if hasRGB {
		clr, err := colors.ParseRGB(data.RGB.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to parse rgb", err.Error())
			return
		}
		hex = clr.ToHEX().String()
	}

	intColor, err := discord.ConvertToInt(hex)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert color", fmt.Sprintf("hex %s: %s", hex, err.Error()))
		return
	}

	data.Dec = types.Int64Value(intColor)
	data.ID = types.StringValue(strconv.FormatInt(intColor, 10))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
