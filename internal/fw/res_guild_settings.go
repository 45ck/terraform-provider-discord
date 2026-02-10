package fw

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/planmod"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewGuildSettingsResource() resource.Resource {
	return &guildSettingsResource{}
}

type guildSettingsResource struct {
	c *discord.RestClient
}

type guildSettingsModel struct {
	ID          types.String `tfsdk:"id"`
	ServerID    types.String `tfsdk:"server_id"`
	PayloadJSON types.String `tfsdk:"payload_json"`
	StateJSON   types.String `tfsdk:"state_json"`
}

func (r *guildSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_guild_settings"
}

func (r *guildSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"server_id": schema.StringAttribute{
				Required: true,
			},
			"payload_json": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					planmod.NormalizeJSONString(),
				},
				Description: "JSON payload to PATCH to /guilds/{guild.id}",
			},
			"state_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized JSON returned from GET /guilds/{guild.id}",
			},
		},
	}
}

func (r *guildSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *guildSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan guildSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := plan.ServerID.ValueString()
	var payload any
	if err := json.Unmarshal([]byte(plan.PayloadJSON.ValueString()), &payload); err != nil {
		resp.Diagnostics.AddError("Invalid payload_json", err.Error())
		return
	}

	var out any
	if err := r.c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s", serverID), nil, payload, &out); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(serverID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	// Refresh computed state_json.
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *guildSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state guildSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readIntoState(ctx, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *guildSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan guildSettingsModel
	var state guildSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serverID := plan.ServerID.ValueString()
	var payload any
	if err := json.Unmarshal([]byte(plan.PayloadJSON.ValueString()), &payload); err != nil {
		resp.Diagnostics.AddError("Invalid payload_json", err.Error())
		return
	}

	var out any
	if err := r.c.DoJSON(ctx, "PATCH", fmt.Sprintf("/guilds/%s", serverID), nil, payload, &out); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = state.ID
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *guildSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No-op; do not attempt to "revert" arbitrary guild settings.
	resp.Diagnostics.AddWarning(
		"discord_guild_settings does not revert guild settings on destroy",
		"Destroying this resource removes it from state only.",
	)
	resp.State.RemoveResource(ctx)
}

func (r *guildSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// ID is the server/guild ID.
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *guildSettingsResource) readIntoState(ctx context.Context, state *guildSettingsModel, diags discordFrameworkDiagnostics) {
	serverID := state.ID.ValueString()
	if serverID == "" {
		serverID = state.ServerID.ValueString()
	}

	var out any
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s", serverID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	b, err := json.Marshal(out)
	if err != nil {
		diags.AddError("JSON error", err.Error())
		return
	}
	norm, err := discord.NormalizeJSON(string(b))
	if err != nil {
		diags.AddError("JSON error", err.Error())
		return
	}

	state.ID = types.StringValue(serverID)
	state.ServerID = types.StringValue(serverID)
	state.StateJSON = types.StringValue(norm)
}

// discordFrameworkDiagnostics is the subset of Diagnostics methods we use across resources.
// It matches *resource.*Response.Diagnostics and *datasource.*Response.Diagnostics.
type discordFrameworkDiagnostics interface {
	AddError(summary, detail string)
	HasError() bool
}
