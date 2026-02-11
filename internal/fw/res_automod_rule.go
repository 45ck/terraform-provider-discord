package fw

import (
	"context"
	"encoding/json"
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

// AutoMod rules have multiple "union" shapes depending on trigger_type.
// Keep a JSON passthrough to avoid pinning users to an incomplete/incorrect schema.
func NewAutoModRuleResource() resource.Resource {
	return &autoModRuleResource{}
}

type autoModRuleResource struct {
	c *discord.RestClient
}

type autoModRuleModel struct {
	ID types.String `tfsdk:"id"`

	ServerID     types.String `tfsdk:"server_id"`
	PayloadJSON  types.String `tfsdk:"payload_json"`
	StateJSON    types.String `tfsdk:"state_json"`
	Reason       types.String `tfsdk:"reason"`
	EffectiveID  types.String `tfsdk:"effective_id"`
	EffectiveGID types.String `tfsdk:"effective_server_id"`
}

type restAutoModRuleLite struct {
	ID string `json:"id"`
}

func (r *autoModRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_automod_rule"
}

func (r *autoModRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"payload_json": schema.StringAttribute{
				Required:    true,
				Description: "JSON payload to POST/PATCH the AutoMod rule.",
				Validators:  []validator.String{validate.JSONString()},
				PlanModifiers: []planmodifier.String{
					planmod.NormalizeJSONString(),
				},
			},
			"state_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized JSON returned from the Discord API for this rule.",
			},
			"reason": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					planmod.IgnoreChangesString(),
				},
				Description: "Optional audit log reason (X-Audit-Log-Reason). This value is not readable.",
			},
			// Convenience computed fields for debugging and composition.
			"effective_id": schema.StringAttribute{Computed: true},
			"effective_server_id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *autoModRuleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *autoModRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan autoModRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var payload any
	if err := json.Unmarshal([]byte(plan.PayloadJSON.ValueString()), &payload); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", err.Error())
		return
	}

	var out restAutoModRuleLite
	if err := r.c.DoJSONWithReason(ctx, "POST", fmt.Sprintf("/guilds/%s/auto-moderation/rules", plan.ServerID.ValueString()), nil, payload, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	if out.ID == "" {
		resp.Diagnostics.AddError("Discord API error", "discord api did not return rule id")
		return
	}

	plan.ID = types.StringValue(out.ID)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *autoModRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state autoModRuleModel
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

func (r *autoModRuleResource) readIntoState(ctx context.Context, state *autoModRuleModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	ruleID := state.ID.ValueString()
	if serverID == "" || ruleID == "" {
		state.ID = types.StringNull()
		return
	}

	var out any
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/auto-moderation/rules/%s", serverID, ruleID), nil, nil, &out); err != nil {
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

	state.StateJSON = types.StringValue(norm)
	state.EffectiveID = types.StringValue(ruleID)
	state.EffectiveGID = types.StringValue(serverID)
}

func (r *autoModRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan autoModRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var payload any
	if err := json.Unmarshal([]byte(plan.PayloadJSON.ValueString()), &payload); err != nil {
		resp.Diagnostics.AddError("Invalid JSON", err.Error())
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/auto-moderation/rules/%s", plan.ServerID.ValueString(), plan.ID.ValueString()), nil, payload, nil, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *autoModRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state autoModRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/auto-moderation/rules/%s", state.ServerID.ValueString(), state.ID.ValueString()), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *autoModRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: server_id:rule_id
	serverID, ruleID, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected server_id:rule_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ruleID)...)
}
