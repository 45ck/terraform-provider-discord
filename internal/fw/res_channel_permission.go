package fw

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewChannelPermissionResource() resource.Resource {
	return &channelPermissionResource{}
}

type channelPermissionResource struct {
	c *discord.RestClient
}

type channelPermissionModel struct {
	ID types.String `tfsdk:"id"`

	ChannelID   types.String `tfsdk:"channel_id"`
	Type        types.String `tfsdk:"type"`
	OverwriteID types.String `tfsdk:"overwrite_id"`

	Allow       types.Int64  `tfsdk:"allow"`
	AllowBits64 types.String `tfsdk:"allow_bits64"`
	Deny        types.Int64  `tfsdk:"deny"`
	DenyBits64  types.String `tfsdk:"deny_bits64"`
}

type restPermOverwriteRead struct {
	ID    string `json:"id"`
	Type  int    `json:"type"` // 0=role, 1=member
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
}

type restChannelPermsRead struct {
	ID                   string                  `json:"id"`
	PermissionOverwrites []restPermOverwriteRead `json:"permission_overwrites"`
}

type restPermOverwriteUpsert struct {
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
	Type  int    `json:"type"`
}

func (r *channelPermissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_permission"
}

func (r *channelPermissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"channel_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.OneOf("ROLE", "USER"),
				},
				Description: "role or user",
			},
			"overwrite_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allow": schema.Int64Attribute{
				Optional: true,
			},
			"allow_bits64": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Allow bitset as 64-bit integer string (decimal or 0x...). Prefer this for newer high-bit permissions.",
			},
			"deny": schema.Int64Attribute{
				Optional: true,
			},
			"deny_bits64": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Deny bitset as 64-bit integer string (decimal or 0x...). Prefer this for newer high-bit permissions.",
			},
		},
	}
}

func (r *channelPermissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func owTypeToIntLegacy(t string) (int, error) {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "role":
		return 0, nil
	case "user":
		return 1, nil
	default:
		return 0, fmt.Errorf("invalid overwrite type %q (expected role or user)", t)
	}
}

func (r *channelPermissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan channelPermissionModel
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

func (r *channelPermissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan channelPermissionModel
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

func (r *channelPermissionResource) upsert(ctx context.Context, plan *channelPermissionModel, diags discordFrameworkDiagnostics) {
	channelID := plan.ChannelID.ValueString()
	overwriteID := plan.OverwriteID.ValueString()

	typ, err := owTypeToIntLegacy(plan.Type.ValueString())
	if err != nil {
		diags.AddError("Invalid type", err.Error())
		return
	}

	// At least one of allow/deny must be provided (by int or bits64).
	hasAny := (!plan.Allow.IsNull()) || (!plan.Deny.IsNull()) ||
		(strings.TrimSpace(plan.AllowBits64.ValueString()) != "") ||
		(strings.TrimSpace(plan.DenyBits64.ValueString()) != "")
	if !hasAny {
		diags.AddError("Invalid configuration", "At least one of allow, deny, allow_bits64, deny_bits64 must be set")
		return
	}

	allow := uint64(0)
	if !plan.Allow.IsNull() {
		allow = uint64(plan.Allow.ValueInt64())
	}
	if s := strings.TrimSpace(plan.AllowBits64.ValueString()); s != "" {
		v, err := discord.Uint64StringToPermissionBit(s)
		if err != nil {
			diags.AddError("Invalid allow_bits64", err.Error())
			return
		}
		allow = v
	}

	deny := uint64(0)
	if !plan.Deny.IsNull() {
		deny = uint64(plan.Deny.ValueInt64())
	}
	if s := strings.TrimSpace(plan.DenyBits64.ValueString()); s != "" {
		v, err := discord.Uint64StringToPermissionBit(s)
		if err != nil {
			diags.AddError("Invalid deny_bits64", err.Error())
			return
		}
		deny = v
	}

	body := restPermOverwriteUpsert{
		Allow: strconv.FormatUint(allow, 10),
		Deny:  strconv.FormatUint(deny, 10),
		Type:  typ,
	}

	if err := r.c.DoJSON(ctx, "PUT", fmt.Sprintf("/channels/%s/permissions/%s", channelID, overwriteID), nil, body, nil); err != nil {
		diags.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(strconv.Itoa(discord.Hashcode(fmt.Sprintf("%s:%s:%s", channelID, overwriteID, strings.ToLower(plan.Type.ValueString())))))
	r.readIntoState(ctx, plan, diags)
}

func (r *channelPermissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state channelPermissionModel
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

func (r *channelPermissionResource) readIntoState(ctx context.Context, state *channelPermissionModel, diags discordFrameworkDiagnostics) {
	channelID := state.ChannelID.ValueString()
	overwriteID := state.OverwriteID.ValueString()
	typ, err := owTypeToIntLegacy(state.Type.ValueString())
	if err != nil {
		diags.AddError("Invalid type", err.Error())
		return
	}

	var ch restChannelPermsRead
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s", channelID), nil, nil, &ch); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	found := false
	for _, x := range ch.PermissionOverwrites {
		if x.Type == typ && x.ID == overwriteID {
			found = true
			state.AllowBits64 = types.StringValue(strings.TrimSpace(x.Allow))
			state.DenyBits64 = types.StringValue(strings.TrimSpace(x.Deny))

			if v, err := discord.Uint64StringToPermissionBit(x.Allow); err == nil {
				if i, err := discord.Uint64ToIntIfFits(v); err == nil {
					state.Allow = types.Int64Value(int64(i))
				} else {
					state.Allow = types.Int64Value(0)
				}
			}
			if v, err := discord.Uint64StringToPermissionBit(x.Deny); err == nil {
				if i, err := discord.Uint64ToIntIfFits(v); err == nil {
					state.Deny = types.Int64Value(int64(i))
				} else {
					state.Deny = types.Int64Value(0)
				}
			}
			break
		}
	}

	if !found {
		state.ID = types.StringNull()
		return
	}

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		state.ID = types.StringValue(strconv.Itoa(discord.Hashcode(fmt.Sprintf("%s:%s:%s", channelID, overwriteID, strings.ToLower(state.Type.ValueString())))))
	}
}

func (r *channelPermissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state channelPermissionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := state.ChannelID.ValueString()
	overwriteID := state.OverwriteID.ValueString()

	if err := r.c.DoJSON(ctx, "DELETE", fmt.Sprintf("/channels/%s/permissions/%s", channelID, overwriteID), nil, nil, nil); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *channelPermissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: channel_id:overwrite_id:type
	parts := strings.SplitN(req.ID, ":", 3)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected channel_id:overwrite_id:type")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("channel_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("overwrite_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(discord.Hashcode(fmt.Sprintf("%s:%s:%s", parts[0], parts[1], strings.ToLower(parts[2])))))...)
}
