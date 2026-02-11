package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/fwutil"
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

func NewGuildTemplateResource() resource.Resource {
	return &guildTemplateResource{}
}

type guildTemplateResource struct {
	c *discord.RestClient
}

type guildTemplateModel struct {
	ID types.String `tfsdk:"id"`

	ServerID    types.String `tfsdk:"server_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Reason      types.String `tfsdk:"reason"`

	UsageCount types.Int64  `tfsdk:"usage_count"`
	IsDirty    types.Bool   `tfsdk:"is_dirty"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
	CreatorID  types.String `tfsdk:"creator_id"`
}

type restGuildTemplate struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`

	UsageCount int64  `json:"usage_count"`
	IsDirty    bool   `json:"is_dirty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`

	CreatorID string `json:"creator_id"`
}

func (r *guildTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_guild_template"
}

func (r *guildTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "Template code."},
			"server_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"reason": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					planmod.IgnoreChangesString(),
				},
				Description: "Optional audit log reason (X-Audit-Log-Reason). This value is not readable.",
			},

			"usage_count": schema.Int64Attribute{
				Computed: true,
			},
			"is_dirty": schema.BoolAttribute{
				Computed: true,
			},
			"created_at": schema.StringAttribute{
				Computed: true,
			},
			"updated_at": schema.StringAttribute{
				Computed: true,
			},
			"creator_id": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *guildTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *guildTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan guildTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}
	if !(plan.Description.IsNull() || plan.Description.IsUnknown()) {
		body["description"] = plan.Description.ValueString()
	}

	var out restGuildTemplate
	if err := r.c.DoJSONWithReason(ctx, "POST", fmt.Sprintf("/guilds/%s/templates", plan.ServerID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(out.Code)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *guildTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state guildTemplateModel
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

func (r *guildTemplateResource) readIntoState(ctx context.Context, state *guildTemplateModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	code := state.ID.ValueString()
	if serverID == "" || code == "" {
		diags.AddError("Invalid state", "server_id and id (template code) must both be set")
		return
	}

	var out []restGuildTemplate
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/templates", serverID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	for _, t := range out {
		if t.Code == code {
			state.ID = types.StringValue(t.Code)
			state.Name = types.StringValue(t.Name)
			state.Description = types.StringValue(t.Description)
			state.UsageCount = types.Int64Value(t.UsageCount)
			state.IsDirty = types.BoolValue(t.IsDirty)
			state.CreatedAt = types.StringValue(t.CreatedAt)
			state.UpdatedAt = types.StringValue(t.UpdatedAt)
			state.CreatorID = types.StringValue(t.CreatorID)
			return
		}
	}

	// Template code not present in this guild anymore.
	state.ID = types.StringNull()
}

func (r *guildTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan guildTemplateModel
	var prior guildTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	code := prior.ID.ValueString()
	body := map[string]any{
		"name": plan.Name.ValueString(),
	}
	if !(plan.Description.IsNull() || plan.Description.IsUnknown()) {
		body["description"] = plan.Description.ValueString()
	}

	var out restGuildTemplate
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/templates/%s", plan.ServerID.ValueString(), code), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(code)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *guildTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state guildTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	code := state.ID.ValueString()
	serverID := state.ServerID.ValueString()
	if code == "" || serverID == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/templates/%s", serverID, code), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *guildTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: "{server_id}:{template_code}".
	serverID, code, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), code)...)
}
