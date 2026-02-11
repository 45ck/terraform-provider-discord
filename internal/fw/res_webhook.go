package fw

import (
	"context"
	"fmt"
	"strings"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/planmod"
	"github.com/aequasi/discord-terraform/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewWebhookResource() resource.Resource {
	return &webhookResource{}
}

type webhookResource struct {
	c *discord.RestClient
}

type restWebhook struct {
	ID        string `json:"id"`
	Type      int    `json:"type"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	URL       string `json:"url"`
}

type webhookModel struct {
	ID types.String `tfsdk:"id"`

	ChannelID     types.String `tfsdk:"channel_id"`
	Name          types.String `tfsdk:"name"`
	AvatarDataURI types.String `tfsdk:"avatar_data_uri"`

	Token   types.String `tfsdk:"token"`
	URL     types.String `tfsdk:"url"`
	GuildID types.String `tfsdk:"guild_id"`

	Reason types.String `tfsdk:"reason"`
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"channel_id": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			// Not readable; keep in state for diff/apply only.
			"avatar_data_uri": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "data: URI for the webhook avatar. Use an empty string to clear.",
			},
			"token": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"guild_id": schema.StringAttribute{
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

func (r *webhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}
	if !plan.AvatarDataURI.IsNull() && !plan.AvatarDataURI.IsUnknown() && strings.TrimSpace(plan.AvatarDataURI.ValueString()) != "" {
		body["avatar"] = plan.AvatarDataURI.ValueString()
	}

	var out restWebhook
	if err := r.c.DoJSONWithReason(ctx, "POST", fmt.Sprintf("/channels/%s/webhooks", plan.ChannelID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(out.ID)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookModel
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

func (r *webhookResource) readIntoState(ctx context.Context, state *webhookModel, diags discordFrameworkDiagnostics) {
	var out restWebhook
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/webhooks/%s", state.ID.ValueString()), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.ID = types.StringValue(out.ID)
	state.ChannelID = types.StringValue(out.ChannelID)
	state.GuildID = types.StringValue(out.GuildID)
	state.Name = types.StringValue(out.Name)

	// Some Discord responses omit token/url on read; preserve if missing.
	if strings.TrimSpace(out.Token) != "" {
		state.Token = types.StringValue(out.Token)
	}
	if strings.TrimSpace(out.URL) != "" {
		state.URL = types.StringValue(out.URL)
	}
}

func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookModel
	var prior webhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
	}

	if plan.ChannelID.ValueString() != prior.ChannelID.ValueString() {
		body["channel_id"] = plan.ChannelID.ValueString()
	}

	if !plan.AvatarDataURI.IsNull() && !plan.AvatarDataURI.IsUnknown() {
		v := strings.TrimSpace(plan.AvatarDataURI.ValueString())
		if v == "" {
			body["avatar"] = nil
		} else {
			body["avatar"] = v
		}
	}

	var out restWebhook
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/webhooks/%s", prior.ID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = prior.ID
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/webhooks/%s", state.ID.ValueString()), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
