package fw

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/aequasi/discord-terraform/internal/fw/fwutil"
	"github.com/aequasi/discord-terraform/internal/fw/planmod"
	"github.com/aequasi/discord-terraform/internal/fw/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewSoundboardSoundResource() resource.Resource {
	return &soundboardSoundResource{}
}

type soundboardSoundResource struct {
	c *discord.RestClient
}

type restSoundboardSoundResource struct {
	Name      string  `json:"name"`
	SoundID   string  `json:"sound_id"`
	Volume    float64 `json:"volume"`
	EmojiID   string  `json:"emoji_id"`
	EmojiName string  `json:"emoji_name"`
	GuildID   string  `json:"guild_id"`
	Available bool    `json:"available"`
}

type soundboardSoundResourceModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String `tfsdk:"server_id"`

	Name    types.String  `tfsdk:"name"`
	Volume  types.Float64 `tfsdk:"volume"`
	EmojiID types.String  `tfsdk:"emoji_id"`

	EmojiName types.String `tfsdk:"emoji_name"`

	SoundFilePath types.String `tfsdk:"sound_file_path"`
	Available     types.Bool   `tfsdk:"available"`

	Reason types.String `tfsdk:"reason"`
}

func (r *soundboardSoundResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_soundboard_sound"
}

func (r *soundboardSoundResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"volume": schema.Float64Attribute{
				Optional: true,
				Default:  float64default.StaticFloat64(1.0),
			},
			"emoji_id": schema.StringAttribute{
				Optional: true,
			},
			"emoji_name": schema.StringAttribute{
				Optional: true,
			},
			// Not readable from the API.
			"sound_file_path": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Path to the sound file. Sent as base64 in the create request.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"available": schema.BoolAttribute{
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

func (r *soundboardSoundResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *soundboardSoundResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg soundboardSoundResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.EmojiID.ValueString() != "" && cfg.EmojiName.ValueString() != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("emoji_id"),
			"Invalid configuration",
			"Only one of emoji_id or emoji_name may be set.",
		)
	}
}

func (r *soundboardSoundResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan soundboardSoundResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	b, err := os.ReadFile(plan.SoundFilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("File error", err.Error())
		return
	}
	snd := base64.StdEncoding.EncodeToString(b)

	body := map[string]any{
		"name":   plan.Name.ValueString(),
		"sound":  snd,
		"volume": plan.Volume.ValueFloat64(),
	}
	if v := plan.EmojiID.ValueString(); v != "" {
		body["emoji_id"] = v
	}
	if v := plan.EmojiName.ValueString(); v != "" {
		body["emoji_name"] = v
	}

	var out restSoundboardSoundResource
	if err := r.c.DoJSONWithReason(ctx, "POST", fmt.Sprintf("/guilds/%s/soundboard-sounds", plan.ServerID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(out.SoundID)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *soundboardSoundResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state soundboardSoundResourceModel
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

func (r *soundboardSoundResource) readIntoState(ctx context.Context, state *soundboardSoundResourceModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	soundID := state.ID.ValueString()
	if serverID == "" || soundID == "" {
		state.ID = types.StringNull()
		return
	}

	var out restSoundboardSoundResource
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/soundboard-sounds/%s", serverID, soundID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.Name = types.StringValue(out.Name)
	state.Volume = types.Float64Value(out.Volume)
	state.EmojiID = types.StringValue(out.EmojiID)
	state.EmojiName = types.StringValue(out.EmojiName)
	state.Available = types.BoolValue(out.Available)
}

func (r *soundboardSoundResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan soundboardSoundResourceModel
	var prior soundboardSoundResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]any{}
	if plan.Name.ValueString() != prior.Name.ValueString() {
		body["name"] = plan.Name.ValueString()
	}
	if !plan.Volume.IsNull() && !prior.Volume.IsNull() && plan.Volume.ValueFloat64() != prior.Volume.ValueFloat64() {
		body["volume"] = plan.Volume.ValueFloat64()
	}
	if plan.EmojiID.ValueString() != prior.EmojiID.ValueString() {
		v := plan.EmojiID.ValueString()
		if v == "" {
			body["emoji_id"] = nil
		} else {
			body["emoji_id"] = v
		}
	}
	if plan.EmojiName.ValueString() != prior.EmojiName.ValueString() {
		v := plan.EmojiName.ValueString()
		if v == "" {
			body["emoji_name"] = nil
		} else {
			body["emoji_name"] = v
		}
	}

	if len(body) > 0 {
		var out restSoundboardSoundResource
		if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/soundboard-sounds/%s", plan.ServerID.ValueString(), plan.ID.ValueString()), nil, body, &out, plan.Reason.ValueString()); err != nil {
			resp.Diagnostics.AddError("Discord API error", err.Error())
			return
		}
	}

	plan.ID = prior.ID
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *soundboardSoundResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state soundboardSoundResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/soundboard-sounds/%s", state.ServerID.ValueString(), state.ID.ValueString()), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *soundboardSoundResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: server_id:sound_id
	serverID, soundID, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected server_id:sound_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), soundID)...)
}
