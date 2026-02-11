package fw

import (
	"context"
	"fmt"

	"github.com/45ck/terraform-provider-discord/discord"
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

func NewWidgetSettingsResource() resource.Resource {
	return &widgetSettingsResource{}
}

type widgetSettingsResource struct {
	c *discord.RestClient
}

type widgetSettingsModel struct {
	ID types.String `tfsdk:"id"`

	ServerID  types.String `tfsdk:"server_id"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	ChannelID types.String `tfsdk:"channel_id"`
	Reason    types.String `tfsdk:"reason"`
}

type restWidgetSettings struct {
	Enabled   bool   `json:"enabled"`
	ChannelID string `json:"channel_id"`
}

func (r *widgetSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_widget_settings"
}

func (r *widgetSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"enabled": schema.BoolAttribute{
				Required: true,
			},
			"channel_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					validate.Snowflake(),
				},
				Description: "Widget channel ID. Required when enabled=true.",
			},
			"reason": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					planmod.IgnoreChangesString(),
				},
				Description: "Optional audit log reason (X-Audit-Log-Reason). This value is not readable.",
			},
		},
	}
}

func (r *widgetSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *widgetSettingsResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg widgetSettingsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.Enabled.IsNull() || cfg.Enabled.IsUnknown() {
		return
	}

	if cfg.Enabled.ValueBool() {
		if cfg.ChannelID.IsNull() || cfg.ChannelID.IsUnknown() || cfg.ChannelID.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(path.Root("channel_id"), "Invalid configuration", "channel_id is required when enabled=true")
		}
	}
}

func (r *widgetSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan widgetSettingsModel
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

func (r *widgetSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state widgetSettingsModel
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

func (r *widgetSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan widgetSettingsModel
	var prior widgetSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = prior.ID
	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *widgetSettingsResource) upsert(ctx context.Context, plan *widgetSettingsModel, diags discordFrameworkDiagnostics) {
	serverID := plan.ServerID.ValueString()

	body := map[string]any{
		"enabled": plan.Enabled.ValueBool(),
	}
	// Only set channel_id when explicitly known. This avoids "clearing" it when config omits it.
	if !(plan.ChannelID.IsNull() || plan.ChannelID.IsUnknown()) && plan.ChannelID.ValueString() != "" {
		body["channel_id"] = plan.ChannelID.ValueString()
	}

	var out restWidgetSettings
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/widget", serverID), nil, body, &out, plan.Reason.ValueString()); err != nil {
		diags.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(serverID)
	plan.ServerID = types.StringValue(serverID)
	plan.Enabled = types.BoolValue(out.Enabled)
	plan.ChannelID = types.StringValue(out.ChannelID)
}

func (r *widgetSettingsResource) readIntoState(ctx context.Context, state *widgetSettingsModel, diags discordFrameworkDiagnostics) {
	serverID := state.ID.ValueString()
	if serverID == "" {
		serverID = state.ServerID.ValueString()
	}

	var out restWidgetSettings
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/widget", serverID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.ID = types.StringValue(serverID)
	state.ServerID = types.StringValue(serverID)
	state.Enabled = types.BoolValue(out.Enabled)
	state.ChannelID = types.StringValue(out.ChannelID)
}

func (r *widgetSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No-op; avoid implicitly changing server settings on destroy.
	resp.Diagnostics.AddWarning(
		"discord_widget_settings does not revert widget settings on destroy",
		"Destroying this resource removes it from state only.",
	)
	resp.State.RemoveResource(ctx)
}

func (r *widgetSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// ID is the server/guild ID.
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
