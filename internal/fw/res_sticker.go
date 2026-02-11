package fw

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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

func NewStickerResource() resource.Resource {
	return &stickerResource{}
}

type stickerResource struct {
	c *discord.RestClient
}

type restSticker struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Tags        string `json:"tags"`
	FormatType  int    `json:"format_type"`
}

type stickerResourceModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String `tfsdk:"server_id"`

	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Tags        types.String `tfsdk:"tags"`

	FilePath types.String `tfsdk:"file_path"`

	FormatType types.Int64  `tfsdk:"format_type"`
	Reason     types.String `tfsdk:"reason"`
}

func (r *stickerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sticker"
}

func (r *stickerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"tags": schema.StringAttribute{
				Required:    true,
				Description: "Sticker tags (comma-separated emoji names).",
			},
			"file_path": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Path to sticker asset (png/apng/lottie json).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"format_type": schema.Int64Attribute{
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
		},
	}
}

func (r *stickerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *stickerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan stickerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p := plan.FilePath.ValueString()
	b, err := os.ReadFile(p)
	if err != nil {
		resp.Diagnostics.AddError("File error", err.Error())
		return
	}

	fields := map[string]string{
		"name":        plan.Name.ValueString(),
		"description": plan.Description.ValueString(),
		"tags":        plan.Tags.ValueString(),
	}

	var out restSticker
	if err := r.c.DoMultipartWithReason(ctx, "POST", fmt.Sprintf("/guilds/%s/stickers", plan.ServerID.ValueString()), nil, fields, "file", filepath.Base(p), b, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(out.ID)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *stickerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state stickerResourceModel
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

func (r *stickerResource) readIntoState(ctx context.Context, state *stickerResourceModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	stickerID := state.ID.ValueString()
	if serverID == "" || stickerID == "" {
		state.ID = types.StringNull()
		return
	}

	var out restSticker
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/stickers/%s", serverID, stickerID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.Name = types.StringValue(out.Name)
	state.Description = types.StringValue(out.Description)
	state.Tags = types.StringValue(out.Tags)
	state.FormatType = types.Int64Value(int64(out.FormatType))
}

func (r *stickerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan stickerResourceModel
	var prior stickerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}
	if plan.Name.ValueString() != prior.Name.ValueString() {
		body["name"] = plan.Name.ValueString()
	}
	if plan.Description.ValueString() != prior.Description.ValueString() {
		body["description"] = plan.Description.ValueString()
	}
	if plan.Tags.ValueString() != prior.Tags.ValueString() {
		body["tags"] = plan.Tags.ValueString()
	}

	if len(body) > 0 {
		var out restSticker
		if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/stickers/%s", plan.ServerID.ValueString(), plan.ID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
			resp.Diagnostics.AddError("Discord API error", err.Error())
			return
		}
	}

	plan.ID = prior.ID
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *stickerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state stickerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/stickers/%s", state.ServerID.ValueString(), state.ID.ValueString()), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *stickerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: server_id:sticker_id
	serverID, stickerID, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected server_id:sticker_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), stickerID)...)
}
