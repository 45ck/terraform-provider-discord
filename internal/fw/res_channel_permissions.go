package fw

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/planmod"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewChannelPermissionsResource() resource.Resource {
	return &channelPermissionsResource{}
}

type channelPermissionsResource struct {
	c *discord.RestClient
}

type channelPermissionsOverwriteModel struct {
	Type        types.String `tfsdk:"type"`
	OverwriteID types.String `tfsdk:"overwrite_id"`
	Allow       types.Int64  `tfsdk:"allow"`
	AllowBits64 types.String `tfsdk:"allow_bits64"`
	Deny        types.Int64  `tfsdk:"deny"`
	DenyBits64  types.String `tfsdk:"deny_bits64"`
}

type channelPermissionsModel struct {
	ID types.String `tfsdk:"id"`

	ChannelID types.String                       `tfsdk:"channel_id"`
	Overwrite []channelPermissionsOverwriteModel `tfsdk:"overwrite"`
	Reason    types.String                       `tfsdk:"reason"`
}

type restPermOverwrite struct {
	ID    string `json:"id"`
	Type  int    `json:"type"` // 0=role, 1=member
	Allow string `json:"allow"`
	Deny  string `json:"deny"`
}

type restChannelOverwrites struct {
	ID                   string              `json:"id"`
	PermissionOverwrites []restPermOverwrite `json:"permission_overwrites"`
}

type owKey struct {
	Type string
	ID   string
}

func (r *channelPermissionsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_channel_permissions"
}

func (r *channelPermissionsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"channel_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"overwrite": schema.SetNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "role or user",
						},
						"overwrite_id": schema.StringAttribute{
							Required: true,
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

func (r *channelPermissionsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func owTypeToInt(t string) (int, error) {
	switch t {
	case "role":
		return 0, nil
	case "user":
		return 1, nil
	default:
		return 0, fmt.Errorf("invalid overwrite type %q (expected role or user)", t)
	}
}

func owTypeFromInt(t int) string {
	switch t {
	case 0:
		return "role"
	case 1:
		return "user"
	default:
		return fmt.Sprintf("%d", t)
	}
}

func normalizeUint64String(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	v, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(v, 10), nil
}

func intToUint64String(v int64) string {
	if v <= 0 {
		return "0"
	}
	return strconv.FormatUint(uint64(v), 10)
}

func desiredOverwrites(d channelPermissionsModel) (map[owKey]map[string]any, error) {
	out := map[owKey]map[string]any{}
	for _, m := range d.Overwrite {
		typ := m.Type.ValueString()
		oid := m.OverwriteID.ValueString()
		ti, err := owTypeToInt(typ)
		if err != nil {
			return nil, err
		}

		allow64, err := normalizeUint64String(m.AllowBits64.ValueString())
		if err != nil {
			return nil, fmt.Errorf("invalid allow_bits64 for overwrite %s: %w", oid, err)
		}
		deny64, err := normalizeUint64String(m.DenyBits64.ValueString())
		if err != nil {
			return nil, fmt.Errorf("invalid deny_bits64 for overwrite %s: %w", oid, err)
		}

		allowStr := allow64
		if allowStr == "" {
			if !m.Allow.IsNull() {
				allowStr = intToUint64String(m.Allow.ValueInt64())
			} else {
				allowStr = "0"
			}
		}
		denyStr := deny64
		if denyStr == "" {
			if !m.Deny.IsNull() {
				denyStr = intToUint64String(m.Deny.ValueInt64())
			} else {
				denyStr = "0"
			}
		}

		out[owKey{Type: typ, ID: oid}] = map[string]any{
			"type":  ti,
			"allow": allowStr,
			"deny":  denyStr,
		}
	}
	return out, nil
}

func readChannelOverwrites(ctx context.Context, c *discord.RestClient, channelID string) (*restChannelOverwrites, error) {
	var out restChannelOverwrites
	if err := c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s", channelID), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *channelPermissionsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan channelPermissionsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.upsert(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(plan.ChannelID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelPermissionsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan channelPermissionsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.upsert(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(plan.ChannelID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *channelPermissionsResource) upsert(ctx context.Context, plan channelPermissionsModel, diags discordFrameworkDiagnostics) {
	channelID := plan.ChannelID.ValueString()
	reason := plan.Reason.ValueString()

	ch, err := readChannelOverwrites(ctx, r.c, channelID)
	if err != nil {
		diags.AddError("Discord API error", err.Error())
		return
	}

	want, err := desiredOverwrites(plan)
	if err != nil {
		diags.AddError("Invalid overwrite", err.Error())
		return
	}

	// Delete existing overwrites not declared.
	for _, ow := range ch.PermissionOverwrites {
		k := owKey{Type: owTypeFromInt(ow.Type), ID: ow.ID}
		if _, ok := want[k]; !ok {
			if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/channels/%s/permissions/%s", channelID, ow.ID), nil, nil, nil, reason); err != nil {
				diags.AddError("Discord API error", err.Error())
				return
			}
		}
	}

	// Upsert desired overwrites.
	for k, body := range want {
		if err := r.c.DoJSONWithReason(ctx, "PUT", fmt.Sprintf("/channels/%s/permissions/%s", channelID, k.ID), nil, body, nil, reason); err != nil {
			diags.AddError("Discord API error", err.Error())
			return
		}
	}
}

func (r *channelPermissionsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state channelPermissionsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	channelID := state.ChannelID.ValueString()
	ch, err := readChannelOverwrites(ctx, r.c, channelID)
	if err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	outs := make([]channelPermissionsOverwriteModel, 0, len(ch.PermissionOverwrites))
	for _, ow := range ch.PermissionOverwrites {
		allowNorm, err := normalizeUint64String(ow.Allow)
		if err != nil && ow.Allow != "" {
			resp.Diagnostics.AddError("Permission parse error", fmt.Sprintf("failed to parse overwrite allow bits for %s: %s", ow.ID, err.Error()))
			return
		}
		denyNorm, err := normalizeUint64String(ow.Deny)
		if err != nil && ow.Deny != "" {
			resp.Diagnostics.AddError("Permission parse error", fmt.Sprintf("failed to parse overwrite deny bits for %s: %s", ow.ID, err.Error()))
			return
		}

		allowInt := int64(0)
		if allowNorm != "" {
			av, _ := strconv.ParseUint(allowNorm, 10, 64)
			if i, ierr := discord.Uint64ToIntIfFits(av); ierr == nil {
				allowInt = int64(i)
			}
		}
		denyInt := int64(0)
		if denyNorm != "" {
			dv, _ := strconv.ParseUint(denyNorm, 10, 64)
			if i, ierr := discord.Uint64ToIntIfFits(dv); ierr == nil {
				denyInt = int64(i)
			}
		}

		outs = append(outs, channelPermissionsOverwriteModel{
			Type:        types.StringValue(owTypeFromInt(ow.Type)),
			OverwriteID: types.StringValue(ow.ID),
			Allow:       types.Int64Value(allowInt),
			AllowBits64: types.StringValue(allowNorm),
			Deny:        types.Int64Value(denyInt),
			DenyBits64:  types.StringValue(denyNorm),
		})
	}

	state.ID = types.StringValue(channelID)
	state.Overwrite = outs
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *channelPermissionsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Authoritative overwrites resource: on destroy, remove only from state.
	resp.Diagnostics.AddWarning("discord_channel_permissions does not revert overwrites on destroy", "Destroying this resource removes it from state only.")
	resp.State.RemoveResource(ctx)
}

func (r *channelPermissionsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("channel_id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
