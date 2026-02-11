package fw

import (
	"context"
	"fmt"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/planmod"
	"github.com/aequasi/discord-terraform/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// discord_server manages an existing guild. Creating guilds is not supported for bot tokens;
// use `terraform import` to adopt a guild and then manage its settings.
func NewServerResource() resource.Resource {
	return &serverResource{}
}

type serverResource struct {
	c *discord.RestClient
}

type serverResourceModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String `tfsdk:"server_id"`

	Name                        types.String `tfsdk:"name"`
	DefaultMessageNotifications types.Int64  `tfsdk:"default_message_notifications"`
	VerificationLevel           types.Int64  `tfsdk:"verification_level"`
	ExplicitContentFilter       types.Int64  `tfsdk:"explicit_content_filter"`

	AfkChannelID types.String `tfsdk:"afk_channel_id"`
	AfkTimeout   types.Int64  `tfsdk:"afk_timeout"`

	IconDataURI   types.String `tfsdk:"icon_data_uri"`
	SplashDataURI types.String `tfsdk:"splash_data_uri"`

	IconHash   types.String `tfsdk:"icon_hash"`
	SplashHash types.String `tfsdk:"splash_hash"`

	OwnerID types.String `tfsdk:"owner_id"`

	Reason types.String `tfsdk:"reason"`
}

func (r *serverResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server"
}

func (r *serverResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"server_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Guild (server) ID.",
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"default_message_notifications": schema.Int64Attribute{
				Optional:    true,
				Default:     int64default.StaticInt64(0),
				Description: "0 = all messages, 1 = only mentions.",
			},
			"verification_level": schema.Int64Attribute{
				Optional:    true,
				Default:     int64default.StaticInt64(0),
				Description: "0..4 depending on guild settings.",
			},
			"explicit_content_filter": schema.Int64Attribute{
				Optional:    true,
				Default:     int64default.StaticInt64(0),
				Description: "0..2 depending on guild settings.",
			},
			"afk_channel_id": schema.StringAttribute{
				Optional:    true,
				Description: "AFK voice channel ID. Use an empty string to clear.",
			},
			"afk_timeout": schema.Int64Attribute{
				Optional:    true,
				Default:     int64default.StaticInt64(300),
				Description: "AFK timeout in seconds.",
			},
			// These are write-only. We persist config in state for drift-free plans; Discord only returns hashes.
			"icon_data_uri": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "data: URI for the server icon. Use an empty string to clear.",
			},
			"splash_data_uri": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "data: URI for the server splash. Use an empty string to clear.",
			},
			"icon_hash": schema.StringAttribute{
				Computed: true,
			},
			"splash_hash": schema.StringAttribute{
				Computed: true,
			},
			"owner_id": schema.StringAttribute{
				Optional:    true,
				Description: "Guild owner ID (transfer). This is a privileged operation and often not permitted for bot tokens.",
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

func (r *serverResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *serverResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Adopt + apply settings.
	var plan serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue(plan.ServerID.ValueString())
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serverResourceModel
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

func (r *serverResource) readIntoState(ctx context.Context, state *serverResourceModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	if serverID == "" {
		serverID = state.ID.ValueString()
	}
	if serverID == "" {
		state.ID = types.StringNull()
		return
	}

	var guild discordRestGuild
	if err := r.c.DoJSON(ctx, "GET", "/guilds/"+serverID, nil, nil, &guild); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.ID = types.StringValue(guild.ID)
	state.ServerID = types.StringValue(guild.ID)
	state.Name = types.StringValue(guild.Name)
	state.DefaultMessageNotifications = types.Int64Value(int64(guild.DefaultMessageNotifications))
	state.VerificationLevel = types.Int64Value(int64(guild.VerificationLevel))
	state.ExplicitContentFilter = types.Int64Value(int64(guild.ExplicitContentFilter))
	state.AfkTimeout = types.Int64Value(int64(guild.AfkTimeout))
	state.IconHash = types.StringValue(guild.Icon)
	state.SplashHash = types.StringValue(guild.Splash)
	if guild.AfkChannelID != "" {
		state.AfkChannelID = types.StringValue(guild.AfkChannelID)
	} else {
		state.AfkChannelID = types.StringNull()
	}
	if guild.OwnerID != "" {
		state.OwnerID = types.StringValue(guild.OwnerID)
	} else {
		state.OwnerID = types.StringNull()
	}

	// Preserve write-only fields in state (icon_data_uri/splash_data_uri) by not touching them here.
}

func (r *serverResource) apply(ctx context.Context, plan *serverResourceModel, diags discordFrameworkDiagnostics) {
	body := map[string]any{
		"name": plan.Name.ValueString(),
	}

	if !plan.VerificationLevel.IsNull() && !plan.VerificationLevel.IsUnknown() {
		body["verification_level"] = int(plan.VerificationLevel.ValueInt64())
	}
	if !plan.DefaultMessageNotifications.IsNull() && !plan.DefaultMessageNotifications.IsUnknown() {
		body["default_message_notifications"] = int(plan.DefaultMessageNotifications.ValueInt64())
	}
	if !plan.ExplicitContentFilter.IsNull() && !plan.ExplicitContentFilter.IsUnknown() {
		body["explicit_content_filter"] = int(plan.ExplicitContentFilter.ValueInt64())
	}
	if !plan.AfkTimeout.IsNull() && !plan.AfkTimeout.IsUnknown() {
		body["afk_timeout"] = int(plan.AfkTimeout.ValueInt64())
	}

	// Empty string clears for IDs and write-only images (translated to JSON null).
	if !plan.AfkChannelID.IsNull() && !plan.AfkChannelID.IsUnknown() {
		v := plan.AfkChannelID.ValueString()
		if v == "" {
			body["afk_channel_id"] = nil
		} else {
			body["afk_channel_id"] = v
		}
	}
	if !plan.OwnerID.IsNull() && !plan.OwnerID.IsUnknown() {
		v := plan.OwnerID.ValueString()
		if v == "" {
			body["owner_id"] = nil
		} else {
			body["owner_id"] = v
		}
	}
	if !plan.IconDataURI.IsNull() && !plan.IconDataURI.IsUnknown() {
		v := plan.IconDataURI.ValueString()
		if v == "" {
			body["icon"] = nil
		} else {
			body["icon"] = v
		}
	}
	if !plan.SplashDataURI.IsNull() && !plan.SplashDataURI.IsUnknown() {
		v := plan.SplashDataURI.ValueString()
		if v == "" {
			body["splash"] = nil
		} else {
			body["splash"] = v
		}
	}

	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s", plan.ServerID.ValueString()), nil, body, nil, plan.Reason.ValueString()); err != nil {
		diags.AddError("Discord API error", err.Error())
		return
	}
}

func (r *serverResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serverResourceModel
	var prior serverResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = prior.ID
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *serverResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("discord_server does not delete the guild on destroy", "Destroying this resource removes it from state only.")
	resp.State.RemoveResource(ctx)
}

func (r *serverResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID is the server_id.
	resource.ImportStatePassthroughID(ctx, path.Root("server_id"), req, resp)
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
