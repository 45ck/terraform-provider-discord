package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/planmod"
	"github.com/aequasi/discord-terraform/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewSystemChannelResource() resource.Resource {
	return &systemChannelResource{}
}

type systemChannelResource struct {
	c *discord.RestClient
}

type systemChannelResourceModel struct {
	ID types.String `tfsdk:"id"`

	ServerID        types.String `tfsdk:"server_id"`
	SystemChannelID types.String `tfsdk:"system_channel_id"`
	Reason          types.String `tfsdk:"reason"`
}

type restGuildForSystemChannel struct {
	ID              string `json:"id"`
	SystemChannelID string `json:"system_channel_id"`
}

func (r *systemChannelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_channel"
}

func (r *systemChannelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"system_channel_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validate.Snowflake(),
				},
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

func (r *systemChannelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *systemChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"system_channel_id": plan.SystemChannelID.ValueString(),
	}
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s", plan.ServerID.ValueString()), nil, body, nil, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.ServerID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemChannelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"system_channel_id": plan.SystemChannelID.ValueString(),
	}
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s", plan.ServerID.ValueString()), nil, body, nil, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(plan.ServerID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemChannelResourceModel
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

func (r *systemChannelResource) readIntoState(ctx context.Context, state *systemChannelResourceModel, diags discordFrameworkDiagnostics) {
	serverID := state.ID.ValueString()
	if serverID == "" {
		serverID = state.ServerID.ValueString()
	}

	var guild restGuildForSystemChannel
	if err := r.c.DoJSON(ctx, "GET", "/guilds/"+serverID, nil, nil, &guild); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.ID = types.StringValue(guild.ID)
	state.ServerID = types.StringValue(guild.ID)
	state.SystemChannelID = types.StringValue(guild.SystemChannelID)
}

func (r *systemChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemChannelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"system_channel_id": nil,
	}
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s", state.ServerID.ValueString()), nil, body, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *systemChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
