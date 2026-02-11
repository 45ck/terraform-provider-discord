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

// discord_guild_template_sync is an "action-style" resource that syncs an existing template
// to the current server state via PUT /guilds/{guild.id}/templates/{template.code}.
//
// This is intentionally separate from discord_guild_template, so you can opt into a periodic sync
// via a user-managed nonce (sync_nonce) without turning the core template resource into an always-on action.
func NewGuildTemplateSyncResource() resource.Resource {
	return &guildTemplateSyncResource{}
}

type guildTemplateSyncResource struct {
	c *discord.RestClient
}

type guildTemplateSyncModel struct {
	ID types.String `tfsdk:"id"`

	ServerID      types.String `tfsdk:"server_id"`
	TemplateCode  types.String `tfsdk:"template_code"`
	SyncNonce     types.String `tfsdk:"sync_nonce"`
	Reason        types.String `tfsdk:"reason"`
	LastUpdatedAt types.String `tfsdk:"updated_at"`
	IsDirty       types.Bool   `tfsdk:"is_dirty"`
}

func (r *guildTemplateSyncResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_guild_template_sync"
}

func (r *guildTemplateSyncResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"template_code": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"sync_nonce": schema.StringAttribute{
				Optional:    true,
				Description: "Change this value to force a resync (Update) without replacing the resource.",
			},
			"reason": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					planmod.IgnoreChangesString(),
				},
				Description: "Optional audit log reason (X-Audit-Log-Reason). This value is not readable.",
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
			"is_dirty": schema.BoolAttribute{
				Computed: true,
			},
		},
	}
}

func (r *guildTemplateSyncResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *guildTemplateSyncResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan guildTemplateSyncModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.sync(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *guildTemplateSyncResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state guildTemplateSyncModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Best-effort: confirm template still exists.
	var out []restGuildTemplate
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/templates", state.ServerID.ValueString()), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	found := false
	for _, t := range out {
		if t.Code == state.TemplateCode.ValueString() {
			state.LastUpdatedAt = types.StringValue(t.UpdatedAt)
			state.IsDirty = types.BoolValue(t.IsDirty)
			found = true
			break
		}
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *guildTemplateSyncResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan guildTemplateSyncModel
	var prior guildTemplateSyncModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = prior.ID
	r.sync(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *guildTemplateSyncResource) sync(ctx context.Context, plan *guildTemplateSyncModel, diags discordFrameworkDiagnostics) {
	serverID := plan.ServerID.ValueString()
	code := plan.TemplateCode.ValueString()

	var out restGuildTemplate
	if err := r.c.DoJSONWithReason(ctx, "PUT", fmt.Sprintf("/guilds/%s/templates/%s", serverID, code), nil, nil, &out, plan.Reason.ValueString()); err != nil {
		diags.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", serverID, code))
	plan.LastUpdatedAt = types.StringValue(out.UpdatedAt)
	plan.IsDirty = types.BoolValue(out.IsDirty)
}

func (r *guildTemplateSyncResource) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No-op. This resource represents the "sync action" only.
	resp.State.RemoveResource(ctx)
}

func (r *guildTemplateSyncResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "{server_id}:{template_code}".
	serverID, code, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template_code"), code)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s:%s", serverID, code))...)
}
