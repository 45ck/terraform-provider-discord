package fw

import (
	"context"
	"encoding/json"
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

// Onboarding schema changes over time; keep a JSON passthrough.
func NewOnboardingResource() resource.Resource {
	return &onboardingResource{}
}

type onboardingResource struct {
	c *discord.RestClient
}

type onboardingModel struct {
	ID types.String `tfsdk:"id"`

	ServerID    types.String `tfsdk:"server_id"`
	PayloadJSON types.String `tfsdk:"payload_json"`
	StateJSON   types.String `tfsdk:"state_json"`
	Reason      types.String `tfsdk:"reason"`
}

func (r *onboardingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_onboarding"
}

func (r *onboardingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Description: "JSON payload to PUT to /guilds/{guild.id}/onboarding",
				Validators:  []validator.String{validate.JSONString()},
				PlanModifiers: []planmodifier.String{
					planmod.NormalizeJSONString(),
				},
			},
			"state_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized JSON returned from GET /guilds/{guild.id}/onboarding",
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

func (r *onboardingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *onboardingResource) upsert(ctx context.Context, plan *onboardingModel, diags discordFrameworkDiagnostics) {
	var payload any
	if err := json.Unmarshal([]byte(plan.PayloadJSON.ValueString()), &payload); err != nil {
		diags.AddError("Invalid JSON", err.Error())
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "PUT", fmt.Sprintf("/guilds/%s/onboarding", plan.ServerID.ValueString()), nil, payload, nil, plan.Reason.ValueString()); err != nil {
		diags.AddError("Discord API error", err.Error())
		return
	}
}

func (r *onboardingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan onboardingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.ServerID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *onboardingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state onboardingModel
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

func (r *onboardingResource) readIntoState(ctx context.Context, state *onboardingModel, diags discordFrameworkDiagnostics) {
	serverID := state.ID.ValueString()
	if serverID == "" {
		serverID = state.ServerID.ValueString()
	}
	if serverID == "" {
		state.ID = types.StringNull()
		return
	}

	var out any
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/onboarding", serverID), nil, nil, &out); err != nil {
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

func (r *onboardingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan onboardingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.ServerID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *onboardingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state onboardingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Best-effort disable. Users that want to "remove" onboarding should explicitly manage enabled=false.
	body := map[string]any{"enabled": false}
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/onboarding", state.ServerID.ValueString()), nil, body, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *onboardingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
