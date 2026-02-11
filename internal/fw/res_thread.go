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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewThreadResource() resource.Resource {
	return &threadResource{}
}

type threadResource struct {
	c *discord.RestClient
}

type restThreadMetadata struct {
	Archived       bool `json:"archived"`
	AutoArchiveDur int  `json:"auto_archive_duration"`
	Locked         bool `json:"locked"`
	Invitable      bool `json:"invitable"`
}

type restThreadChannel struct {
	ID             string              `json:"id"`
	GuildID        string              `json:"guild_id"`
	ParentID       string              `json:"parent_id"`
	Name           string              `json:"name"`
	Type           uint                `json:"type"`
	RateLimit      int                 `json:"rate_limit_per_user"`
	ThreadMetadata *restThreadMetadata `json:"thread_metadata"`
	AppliedTags    []string            `json:"applied_tags"`
}

type threadModel struct {
	ID types.String `tfsdk:"id"`

	ChannelID types.String `tfsdk:"channel_id"`
	MessageID types.String `tfsdk:"message_id"`

	Type types.String `tfsdk:"type"`
	Name types.String `tfsdk:"name"`

	AutoArchiveDuration types.Int64 `tfsdk:"auto_archive_duration"`
	Invitable           types.Bool  `tfsdk:"invitable"`
	RateLimitPerUser    types.Int64 `tfsdk:"rate_limit_per_user"`
	Archived            types.Bool  `tfsdk:"archived"`
	Locked              types.Bool  `tfsdk:"locked"`

	AppliedTags types.Set `tfsdk:"applied_tags"`

	ServerID types.String `tfsdk:"server_id"`

	Content types.String       `tfsdk:"content"`
	Embed   *messageEmbedModel `tfsdk:"embed"`
	Reason  types.String       `tfsdk:"reason"`
}

func (r *threadResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_thread"
}

func (r *threadResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"channel_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Parent channel ID (text/news/forum/media).",
			},
			"message_id": schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "If set, starts a thread from an existing message.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Default:     stringdefault.StaticString("public_thread"),
				Description: "Thread type: public_thread, private_thread, announcement_thread.",
				Validators: []validator.String{
					validate.OneOf("PUBLIC_THREAD", "PRIVATE_THREAD", "ANNOUNCEMENT_THREAD"),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"auto_archive_duration": schema.Int64Attribute{
				Optional:    true,
				Description: "Minutes: 60, 1440, 4320, 10080.",
			},
			"invitable": schema.BoolAttribute{
				Optional:    true,
				Description: "Private thread invite permission.",
			},
			"rate_limit_per_user": schema.Int64Attribute{
				Optional:    true,
				Description: "Thread slowmode in seconds.",
			},
			"archived": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether the thread is archived.",
			},
			"locked": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether the thread is locked.",
			},
			"applied_tags": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Forum/media thread tags (IDs).",
			},
			"server_id": schema.StringAttribute{
				Computed: true,
			},
			"content": schema.StringAttribute{
				Optional:    true,
				Description: "Initial message content (forum/media threads).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					planmod.TrimTrailingCRLFString(),
				},
			},
			"embed": schema.SingleNestedAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"title":       schema.StringAttribute{Optional: true},
					"description": schema.StringAttribute{Optional: true},
					"url":         schema.StringAttribute{Optional: true},
					"timestamp":   schema.StringAttribute{Optional: true},
					"color":       schema.Int64Attribute{Optional: true},
					"footer": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"text":     schema.StringAttribute{Required: true},
							"icon_url": schema.StringAttribute{Optional: true},
						},
					},
					"image": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"url":       schema.StringAttribute{Required: true},
							"proxy_url": schema.StringAttribute{Computed: true},
							"height":    schema.Int64Attribute{Optional: true},
							"width":     schema.Int64Attribute{Optional: true},
						},
					},
					"thumbnail": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"url":       schema.StringAttribute{Required: true},
							"proxy_url": schema.StringAttribute{Computed: true},
							"height":    schema.Int64Attribute{Optional: true},
							"width":     schema.Int64Attribute{Optional: true},
						},
					},
					"video": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"url":    schema.StringAttribute{Required: true},
							"height": schema.Int64Attribute{Optional: true},
							"width":  schema.Int64Attribute{Optional: true},
						},
					},
					"provider": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{Optional: true},
							"url":  schema.StringAttribute{Optional: true},
						},
					},
					"author": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"name":           schema.StringAttribute{Optional: true},
							"url":            schema.StringAttribute{Optional: true},
							"icon_url":       schema.StringAttribute{Optional: true},
							"proxy_icon_url": schema.StringAttribute{Computed: true},
						},
					},
					"fields": schema.ListNestedAttribute{
						Optional: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name":   schema.StringAttribute{Required: true},
								"value":  schema.StringAttribute{Optional: true},
								"inline": schema.BoolAttribute{Optional: true},
							},
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
				Description: "Optional audit log reason (X-Audit-Log-Reason). This value is not readable.",
			},
		},
	}
}

func (r *threadResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *threadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan threadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parentID := plan.ChannelID.ValueString()

	t := plan.Type.ValueString()
	if strings.TrimSpace(t) == "" {
		t = "public_thread"
	}

	typ, ok := discord.GetDiscordChannelType(t)
	if !ok {
		resp.Diagnostics.AddError("Invalid type", fmt.Sprintf("unsupported thread type: %s", t))
		return
	}

	body := map[string]any{
		"name": plan.Name.ValueString(),
		"type": int(typ),
	}
	if !plan.AutoArchiveDuration.IsNull() && !plan.AutoArchiveDuration.IsUnknown() {
		body["auto_archive_duration"] = int(plan.AutoArchiveDuration.ValueInt64())
	}
	if !plan.Invitable.IsNull() && !plan.Invitable.IsUnknown() {
		body["invitable"] = plan.Invitable.ValueBool()
	}
	if !plan.RateLimitPerUser.IsNull() && !plan.RateLimitPerUser.IsUnknown() {
		body["rate_limit_per_user"] = int(plan.RateLimitPerUser.ValueInt64())
	}
	if !plan.AppliedTags.IsNull() && !plan.AppliedTags.IsUnknown() {
		tags := []string{}
		resp.Diagnostics.Append(plan.AppliedTags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["applied_tags"] = tags
	}

	// Optional initial message (forum/media).
	if !plan.Content.IsNull() && !plan.Content.IsUnknown() && plan.Content.ValueString() != "" {
		msg := map[string]any{"content": plan.Content.ValueString()}
		if plan.Embed != nil {
			msg["embeds"] = []restEmbed{embedToRest(plan.Embed)}
		}
		body["message"] = msg
	} else if plan.Embed != nil {
		body["message"] = map[string]any{"embeds": []restEmbed{embedToRest(plan.Embed)}}
	}

	var out restThreadChannel
	if !plan.MessageID.IsNull() && !plan.MessageID.IsUnknown() && plan.MessageID.ValueString() != "" {
		path := fmt.Sprintf("/channels/%s/messages/%s/threads", parentID, plan.MessageID.ValueString())
		if err := r.c.DoJSONWithReason(ctx, "POST", path, nil, body, &out, plan.Reason.ValueString()); err != nil {
			resp.Diagnostics.AddError("Discord API error", err.Error())
			return
		}
	} else {
		path := fmt.Sprintf("/channels/%s/threads", parentID)
		if err := r.c.DoJSONWithReason(ctx, "POST", path, nil, body, &out, plan.Reason.ValueString()); err != nil {
			resp.Diagnostics.AddError("Discord API error", err.Error())
			return
		}
	}

	plan.ID = types.StringValue(out.ID)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *threadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state threadModel
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

func (r *threadResource) readIntoState(ctx context.Context, state *threadModel, diags discordFrameworkDiagnostics) {
	var out restThreadChannel
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s", state.ID.ValueString()), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	tt, ok := discord.GetTextChannelType(out.Type)
	if ok {
		state.Type = types.StringValue(tt)
	}

	state.ServerID = types.StringValue(out.GuildID)
	state.ChannelID = types.StringValue(out.ParentID)
	state.Name = types.StringValue(out.Name)
	state.RateLimitPerUser = types.Int64Value(int64(out.RateLimit))

	if out.ThreadMetadata != nil {
		state.Archived = types.BoolValue(out.ThreadMetadata.Archived)
		state.Locked = types.BoolValue(out.ThreadMetadata.Locked)
		state.Invitable = types.BoolValue(out.ThreadMetadata.Invitable)
		state.AutoArchiveDuration = types.Int64Value(int64(out.ThreadMetadata.AutoArchiveDur))
	}

	if out.AppliedTags != nil {
		tags, tdiags := types.SetValueFrom(ctx, types.StringType, out.AppliedTags)
		if tdiags.HasError() {
			diags.AddError("State error", "failed to set applied_tags")
			return
		}
		state.AppliedTags = tags
	} else {
		state.AppliedTags = types.SetNull(types.StringType)
	}
}

func (r *threadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan threadModel
	var prior threadModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}

	if plan.Name.ValueString() != prior.Name.ValueString() {
		body["name"] = plan.Name.ValueString()
	}
	if !plan.RateLimitPerUser.IsNull() && !prior.RateLimitPerUser.IsNull() && plan.RateLimitPerUser.ValueInt64() != prior.RateLimitPerUser.ValueInt64() {
		body["rate_limit_per_user"] = int(plan.RateLimitPerUser.ValueInt64())
	}
	if !plan.Archived.IsNull() && !prior.Archived.IsNull() && plan.Archived.ValueBool() != prior.Archived.ValueBool() {
		body["archived"] = plan.Archived.ValueBool()
	}
	if !plan.Locked.IsNull() && !prior.Locked.IsNull() && plan.Locked.ValueBool() != prior.Locked.ValueBool() {
		body["locked"] = plan.Locked.ValueBool()
	}
	if !plan.AutoArchiveDuration.IsNull() && !prior.AutoArchiveDuration.IsNull() && plan.AutoArchiveDuration.ValueInt64() != prior.AutoArchiveDuration.ValueInt64() {
		body["auto_archive_duration"] = int(plan.AutoArchiveDuration.ValueInt64())
	}
	if !plan.Invitable.IsNull() && !prior.Invitable.IsNull() && plan.Invitable.ValueBool() != prior.Invitable.ValueBool() {
		body["invitable"] = plan.Invitable.ValueBool()
	}
	if !plan.AppliedTags.IsNull() && !plan.AppliedTags.IsUnknown() {
		tags := []string{}
		resp.Diagnostics.Append(plan.AppliedTags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["applied_tags"] = tags
	}

	if len(body) > 0 {
		if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/channels/%s", prior.ID.ValueString()), nil, body, nil, plan.Reason.ValueString()); err != nil {
			resp.Diagnostics.AddError("Discord API error", err.Error())
			return
		}
	}

	plan.ID = prior.ID
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *threadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state threadModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/channels/%s", state.ID.ValueString()), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *threadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
