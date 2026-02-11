package fw

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/45ck/terraform-provider-discord/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewAPIRequestDataSource() datasource.DataSource {
	return &apiRequestDataSource{}
}

type apiRequestDataSource struct {
	c *discord.RestClient
}

type apiRequestModel struct {
	ID           types.String `tfsdk:"id"`
	Path         types.String `tfsdk:"path"`
	QueryJSON    types.String `tfsdk:"query_json"`
	ResponseJSON types.String `tfsdk:"response_json"`
}

func (d *apiRequestDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_request"
}

func (d *apiRequestDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "API path starting with '/'. Example: '/guilds/{guild_id}/channels'.",
			},
			"query_json": schema.StringAttribute{
				Optional:    true,
				Description: "JSON object encoded query params. Example: jsonencode({ limit = 100 })",
			},
			"response_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized JSON response body.",
			},
		},
	}
}

func (d *apiRequestDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *apiRequestDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data apiRequestModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pathStr := data.Path.ValueString()

	var q map[string]interface{}
	if v := data.QueryJSON.ValueString(); v != "" {
		if err := json.Unmarshal([]byte(v), &q); err != nil {
			resp.Diagnostics.AddError("Invalid query_json", err.Error())
			return
		}
	}

	query := url.Values{}
	for k, v := range q {
		switch t := v.(type) {
		case string:
			query.Set(k, t)
		case bool:
			if t {
				query.Set(k, "true")
			} else {
				query.Set(k, "false")
			}
		case float64:
			query.Set(k, strconv.FormatFloat(t, 'f', -1, 64))
		case []interface{}:
			for _, item := range t {
				query.Add(k, stringifyQueryValue(item))
			}
		default:
			b, _ := json.Marshal(t)
			query.Set(k, string(b))
		}
	}

	var out interface{}
	if err := d.c.DoJSON(ctx, "GET", pathStr, query, nil, &out); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	b, err := json.Marshal(out)
	if err != nil {
		resp.Diagnostics.AddError("JSON error", err.Error())
		return
	}
	norm, err := discord.NormalizeJSON(string(b))
	if err != nil {
		resp.Diagnostics.AddError("JSON error", err.Error())
		return
	}

	data.ID = types.StringValue(strconv.Itoa(discord.Hashcode(pathStr + "|" + norm)))
	data.ResponseJSON = types.StringValue(norm)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func stringifyQueryValue(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}
