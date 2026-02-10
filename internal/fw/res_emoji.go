package fw

import (
	"context"
	"fmt"
	"strings"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/fwutil"
	"github.com/aequasi/discord-terraform/internal/fw/planmod"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewEmojiResource() resource.Resource {
	return &emojiResource{}
}

type emojiResource struct {
	c *discord.RestClient
}

type restEmoji struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Roles    []string `json:"roles"`
	Managed  bool     `json:"managed"`
	Animated bool     `json:"animated"`
}

type emojiResourceModel struct {
	ID types.String `tfsdk:"id"`

	ServerID      types.String `tfsdk:"server_id"`
	Name          types.String `tfsdk:"name"`
	ImageDataURI  types.String `tfsdk:"image_data_uri"`
	Roles         types.Set    `tfsdk:"roles"`
	Managed       types.Bool   `tfsdk:"managed"`
	Animated      types.Bool   `tfsdk:"animated"`
	Reason        types.String `tfsdk:"reason"`
	EffectiveName types.String `tfsdk:"effective_name"`
}

func (r *emojiResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_emoji"
}

func (r *emojiResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			// This value cannot be read back from the Discord API. Make it optional so
			// existing emojis can be imported/managed without forcing replacement.
			"image_data_uri": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "data: URI for the emoji image. Required on create; changing forces replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"roles": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Role IDs allowed to use this emoji. Empty means usable by everyone.",
			},
			"managed": schema.BoolAttribute{
				Computed: true,
			},
			"animated": schema.BoolAttribute{
				Computed: true,
			},
			"reason": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					planmod.IgnoreChangesString(),
				},
				Description: "Optional audit log reason (X-Audit-Log-Reason). This value is not readable.",
			},
			// Convenience: often used for naming references in other resources; mirrors `name`.
			"effective_name": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *emojiResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *emojiResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan emojiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ImageDataURI.IsNull() || plan.ImageDataURI.IsUnknown() || strings.TrimSpace(plan.ImageDataURI.ValueString()) == "" {
		resp.Diagnostics.AddError("Invalid configuration", "image_data_uri must be set when creating an emoji")
		return
	}

	roles := []string{}
	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	body := map[string]any{
		"name":  plan.Name.ValueString(),
		"image": plan.ImageDataURI.ValueString(),
		"roles": roles,
	}

	var out restEmoji
	if err := r.c.DoJSONWithReason(ctx, "POST", fmt.Sprintf("/guilds/%s/emojis", plan.ServerID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(out.ID)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *emojiResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state emojiResourceModel
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

func (r *emojiResource) readIntoState(ctx context.Context, state *emojiResourceModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	emojiID := state.ID.ValueString()
	if serverID == "" || emojiID == "" {
		state.ID = types.StringNull()
		return
	}

	var out restEmoji
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/emojis/%s", serverID, emojiID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	vals := make([]attr.Value, 0, len(out.Roles))
	for _, rid := range out.Roles {
		vals = append(vals, types.StringValue(rid))
	}

	state.ID = types.StringValue(out.ID)
	state.ServerID = types.StringValue(serverID)
	state.Name = types.StringValue(out.Name)
	state.Roles = types.SetValueMust(types.StringType, vals)
	state.Managed = types.BoolValue(out.Managed)
	state.Animated = types.BoolValue(out.Animated)
	state.EffectiveName = types.StringValue(out.Name)
}

func (r *emojiResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan emojiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles := []string{}
	if !plan.Roles.IsNull() && !plan.Roles.IsUnknown() {
		resp.Diagnostics.Append(plan.Roles.ElementsAs(ctx, &roles, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	body := map[string]any{
		"name":  plan.Name.ValueString(),
		"roles": roles,
	}

	var out restEmoji
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/emojis/%s", plan.ServerID.ValueString(), plan.ID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(out.ID)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *emojiResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state emojiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/emojis/%s", state.ServerID.ValueString(), state.ID.ValueString()), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *emojiResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: server_id:emoji_id
	serverID, emojiID, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected server_id:emoji_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), emojiID)...)
}
