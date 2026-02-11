package fw

import (
	"context"
	"fmt"

	"github.com/45ck/terraform-provider-discord/discord"
	"github.com/45ck/terraform-provider-discord/internal/fw/fwutil"
	"github.com/45ck/terraform-provider-discord/internal/fw/planmod"
	"github.com/45ck/terraform-provider-discord/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewMemberNicknameResource() resource.Resource {
	return &memberNicknameResource{}
}

type memberNicknameResource struct {
	c *discord.RestClient
}

type restGuildMember struct {
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	CommunicationDisabledUntil string `json:"communication_disabled_until"`
	Nick                       string `json:"nick"`
}

type memberNicknameModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String `tfsdk:"server_id"`
	UserID   types.String `tfsdk:"user_id"`
	Nick     types.String `tfsdk:"nick"`
	Reason   types.String `tfsdk:"reason"`
}

func (r *memberNicknameResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_member_nickname"
}

func (r *memberNicknameResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"user_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"nick": schema.StringAttribute{
				Required:    true,
				Description: "Nickname for the member. Use an empty string to clear.",
			},
			"reason": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					planmod.IgnoreChangesString(),
				},
			},
		},
	}
}

func (r *memberNicknameResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *memberNicknameResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan memberNicknameModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *memberNicknameResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan memberNicknameModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *memberNicknameResource) upsert(ctx context.Context, plan *memberNicknameModel, diags discordFrameworkDiagnostics) {
	serverID := plan.ServerID.ValueString()
	userID := plan.UserID.ValueString()

	nick := plan.Nick.ValueString()
	var val any
	if nick == "" {
		val = nil
	} else {
		val = nick
	}
	body := map[string]any{"nick": val}

	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/members/%s", serverID, userID), nil, body, nil, plan.Reason.ValueString()); err != nil {
		diags.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", serverID, userID))
	r.readIntoState(ctx, plan, diags)
}

func (r *memberNicknameResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state memberNicknameModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readIntoState(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.ID.IsNull() {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *memberNicknameResource) readIntoState(ctx context.Context, state *memberNicknameModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	userID := state.UserID.ValueString()
	if serverID == "" || userID == "" {
		sid, uid, err := fwutil.ParseTwoIDs(state.ID.ValueString())
		if err != nil {
			diags.AddError("Invalid ID", err.Error())
			return
		}
		serverID, userID = sid, uid
	}

	var out restGuildMember
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/members/%s", serverID, userID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s:%s", serverID, userID))
	state.ServerID = types.StringValue(serverID)
	state.UserID = types.StringValue(userID)
	state.Nick = types.StringValue(out.Nick)
}

func (r *memberNicknameResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state memberNicknameModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := state.ServerID.ValueString()
	userID := state.UserID.ValueString()

	body := map[string]any{"nick": nil}
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/members/%s", serverID, userID), nil, body, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *memberNicknameResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serverID, userID, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s:%s", serverID, userID))...)
}
