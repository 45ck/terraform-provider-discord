package fw

import (
	"context"

	"github.com/45ck/terraform-provider-discord/discord"
	"github.com/45ck/terraform-provider-discord/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewMemberDataSource() datasource.DataSource {
	return &memberDataSource{}
}

type memberDataSource struct {
	c *discord.RestClient
}

type restUserDS struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
}

type restMemberDS struct {
	User         restUserDS `json:"user"`
	Nick         string     `json:"nick"`
	Roles        []string   `json:"roles"`
	JoinedAt     string     `json:"joined_at"`
	PremiumSince string     `json:"premium_since"`
}

type memberModel struct {
	ID            types.String `tfsdk:"id"`
	ServerID      types.String `tfsdk:"server_id"`
	UserID        types.String `tfsdk:"user_id"`
	Username      types.String `tfsdk:"username"`
	Discriminator types.String `tfsdk:"discriminator"`
	JoinedAt      types.String `tfsdk:"joined_at"`
	PremiumSince  types.String `tfsdk:"premium_since"`
	Avatar        types.String `tfsdk:"avatar"`
	Nick          types.String `tfsdk:"nick"`
	Roles         types.Set    `tfsdk:"roles"`
	InServer      types.Bool   `tfsdk:"in_server"`
}

func (d *memberDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_member"
}

func (d *memberDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"user_id": schema.StringAttribute{
				Optional:    true,
				Description: "Prefer user_id. Username-based lookup is not supported.",
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"username": schema.StringAttribute{Optional: true, Description: "Not supported for bot tokens; use user_id."},
			"discriminator": schema.StringAttribute{
				Optional:    true,
				Description: "Not supported for bot tokens; use user_id.",
			},
			"joined_at":     schema.StringAttribute{Computed: true},
			"premium_since": schema.StringAttribute{Computed: true},
			"avatar":        schema.StringAttribute{Computed: true},
			"nick":          schema.StringAttribute{Computed: true},
			"roles":         schema.SetAttribute{Computed: true, ElementType: types.StringType},
			"in_server":     schema.BoolAttribute{Computed: true},
		},
	}
}

func (d *memberDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *memberDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data memberModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Username-based lookup is not supported (requires member listing/search, which is not scalable).
	if !data.Username.IsNull() && data.Username.ValueString() != "" {
		resp.Diagnostics.AddError("Unsupported lookup", "discord_member data source lookup by username/discriminator is not supported; use user_id")
		return
	}

	serverID := data.ServerID.ValueString()
	userID := data.UserID.ValueString()
	if userID == "" {
		resp.Diagnostics.AddError("Missing user_id", "either user_id or username must be set (user_id required for bot tokens)")
		return
	}

	var member restMemberDS
	err := d.c.DoJSON(ctx, "GET", "/guilds/"+serverID+"/members/"+userID, nil, nil, &member)
	if err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			data.ID = types.StringValue(userID)
			data.InServer = types.BoolValue(false)
			data.JoinedAt = types.StringNull()
			data.PremiumSince = types.StringNull()
			data.Roles = types.SetNull(types.StringType)
			data.Username = types.StringNull()
			data.Discriminator = types.StringNull()
			data.Avatar = types.StringNull()
			data.Nick = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	data.ID = types.StringValue(member.User.ID)
	data.InServer = types.BoolValue(true)
	data.JoinedAt = types.StringValue(member.JoinedAt)
	if member.PremiumSince != "" {
		data.PremiumSince = types.StringValue(member.PremiumSince)
	} else {
		data.PremiumSince = types.StringNull()
	}
	data.Username = types.StringValue(member.User.Username)
	data.Discriminator = types.StringValue(member.User.Discriminator)
	data.Avatar = types.StringValue(member.User.Avatar)
	data.Nick = types.StringValue(member.Nick)

	roles, diags := types.SetValueFrom(ctx, types.StringType, member.Roles)
	resp.Diagnostics.Append(diags...)
	data.Roles = roles

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
