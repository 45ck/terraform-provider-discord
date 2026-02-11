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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewScheduledEventResource() resource.Resource {
	return &scheduledEventResource{}
}

type scheduledEventResource struct {
	c *discord.RestClient
}

type restScheduledEvent struct {
	ID                 string                      `json:"id"`
	GuildID            string                      `json:"guild_id"`
	ChannelID          string                      `json:"channel_id"`
	Name               string                      `json:"name"`
	Description        string                      `json:"description"`
	ScheduledStartTime string                      `json:"scheduled_start_time"`
	ScheduledEndTime   string                      `json:"scheduled_end_time"`
	PrivacyLevel       int                         `json:"privacy_level"`
	Status             int                         `json:"status"`
	EntityType         int                         `json:"entity_type"`
	EntityMetadata     *restScheduledEventMetadata `json:"entity_metadata"`
	Image              string                      `json:"image"`
}

type restScheduledEventMetadata struct {
	Location string `json:"location"`
}

type scheduledEventModel struct {
	ID types.String `tfsdk:"id"`

	ServerID types.String `tfsdk:"server_id"`

	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	ScheduledStartTime types.String `tfsdk:"scheduled_start_time"`
	ScheduledEndTime   types.String `tfsdk:"scheduled_end_time"`
	PrivacyLevel       types.Int64  `tfsdk:"privacy_level"`
	EntityType         types.Int64  `tfsdk:"entity_type"`
	ChannelID          types.String `tfsdk:"channel_id"`
	Location           types.String `tfsdk:"location"`

	ImageDataURI types.String `tfsdk:"image_data_uri"`
	ImageHash    types.String `tfsdk:"image_hash"`

	Status types.Int64  `tfsdk:"status"`
	Reason types.String `tfsdk:"reason"`
}

func (r *scheduledEventResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_event"
}

func (r *scheduledEventResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"scheduled_start_time": schema.StringAttribute{
				Required:    true,
				Description: "RFC3339 timestamp",
			},
			"scheduled_end_time": schema.StringAttribute{
				Optional:    true,
				Description: "RFC3339 timestamp (required for external events)",
			},
			"privacy_level": schema.Int64Attribute{
				Optional: true,
				Default:  int64default.StaticInt64(2),
			},
			"entity_type": schema.Int64Attribute{
				Required:    true,
				Description: "1=stage instance, 2=voice, 3=external",
			},
			"channel_id": schema.StringAttribute{
				Optional:    true,
				Description: "Required for stage/voice events",
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"location": schema.StringAttribute{
				Optional:    true,
				Description: "External event location (entity_metadata.location)",
			},
			// Not readable; keep in state for diff/apply only.
			"image_data_uri": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "data: URI for event cover image",
			},
			"image_hash": schema.StringAttribute{
				Computed:    true,
				Description: "Image hash returned by Discord for the current cover image.",
			},
			"status": schema.Int64Attribute{
				Optional:    true,
				Description: "Event status (set on update to start/end/cancel where supported).",
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

func (r *scheduledEventResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *scheduledEventResource) payload(plan *scheduledEventModel, includeStatus bool) map[string]any {
	body := map[string]any{
		"name":                 plan.Name.ValueString(),
		"scheduled_start_time": plan.ScheduledStartTime.ValueString(),
		"privacy_level":        int(plan.PrivacyLevel.ValueInt64()),
		"entity_type":          int(plan.EntityType.ValueInt64()),
	}
	if v := plan.Description.ValueString(); v != "" {
		body["description"] = v
	}
	if v := plan.ScheduledEndTime.ValueString(); v != "" {
		body["scheduled_end_time"] = v
	}
	if v := plan.ChannelID.ValueString(); v != "" {
		body["channel_id"] = v
	}
	if v := plan.Location.ValueString(); v != "" {
		body["entity_metadata"] = map[string]any{"location": v}
	}
	if v := plan.ImageDataURI.ValueString(); v != "" {
		body["image"] = v
	}
	if includeStatus && !plan.Status.IsNull() && !plan.Status.IsUnknown() {
		body["status"] = int(plan.Status.ValueInt64())
	}
	return body
}

func (r *scheduledEventResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scheduledEventModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var out restScheduledEvent
	if err := r.c.DoJSONWithReason(ctx, "POST", fmt.Sprintf("/guilds/%s/scheduled-events", plan.ServerID.ValueString()), nil, r.payload(&plan, false), &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(out.ID)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *scheduledEventResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scheduledEventModel
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

func (r *scheduledEventResource) readIntoState(ctx context.Context, state *scheduledEventModel, diags discordFrameworkDiagnostics) {
	serverID := state.ServerID.ValueString()
	eventID := state.ID.ValueString()
	if serverID == "" || eventID == "" {
		state.ID = types.StringNull()
		return
	}

	var out restScheduledEvent
	if err := r.c.DoJSON(ctx, "GET", fmt.Sprintf("/guilds/%s/scheduled-events/%s", serverID, eventID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.ID = types.StringValue(out.ID)
	state.ServerID = types.StringValue(serverID)
	state.Name = types.StringValue(out.Name)
	state.Description = types.StringValue(out.Description)
	state.ScheduledStartTime = types.StringValue(out.ScheduledStartTime)
	state.ScheduledEndTime = types.StringValue(out.ScheduledEndTime)
	state.PrivacyLevel = types.Int64Value(int64(out.PrivacyLevel))
	state.EntityType = types.Int64Value(int64(out.EntityType))
	state.ChannelID = types.StringValue(out.ChannelID)
	if out.EntityMetadata != nil {
		state.Location = types.StringValue(out.EntityMetadata.Location)
	} else {
		state.Location = types.StringNull()
	}
	state.ImageHash = types.StringValue(out.Image)
	state.Status = types.Int64Value(int64(out.Status))
}

func (r *scheduledEventResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scheduledEventModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var out restScheduledEvent
	if err := r.c.DoJSONWithReason(ctx, "PATCH", fmt.Sprintf("/guilds/%s/scheduled-events/%s", plan.ServerID.ValueString(), plan.ID.ValueString()), nil, r.payload(&plan, true), &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(out.ID)
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *scheduledEventResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scheduledEventModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.c.DoJSONWithReason(ctx, "DELETE", fmt.Sprintf("/guilds/%s/scheduled-events/%s", state.ServerID.ValueString(), state.ID.ValueString()), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *scheduledEventResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: server_id:event_id
	serverID, eventID, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected server_id:event_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_id"), serverID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), eventID)...)
}
