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

func NewThreadMemberResource() resource.Resource {
	return &threadMemberResource{}
}

type threadMemberResource struct {
	c *discord.RestClient
}

type restThreadMemberResource struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	JoinTimestamp string `json:"join_timestamp"`
	Flags         int    `json:"flags"`
}

type threadMemberResourceModel struct {
	ID types.String `tfsdk:"id"`

	ThreadID types.String `tfsdk:"thread_id"`
	UserID   types.String `tfsdk:"user_id"`

	Reason types.String `tfsdk:"reason"`

	JoinTimestamp types.String `tfsdk:"join_timestamp"`
	Flags         types.Int64  `tfsdk:"flags"`
}

func (r *threadMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_thread_member"
}

func (r *threadMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
			"thread_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Thread channel ID.",
				Validators: []validator.String{
					validate.Snowflake(),
				},
			},
			"user_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "User ID, or @me for the bot.",
				Validators: []validator.String{
					validate.SnowflakeOrAtMe(),
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
			"join_timestamp": schema.StringAttribute{Computed: true},
			"flags":          schema.Int64Attribute{Computed: true},
		},
	}
}

func (r *threadMemberResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func threadMemberPath(threadID, userID string) string {
	if userID == "@me" {
		return fmt.Sprintf("/channels/%s/thread-members/@me", threadID)
	}
	return fmt.Sprintf("/channels/%s/thread-members/%s", threadID, userID)
}

func (r *threadMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan threadMemberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	threadID := plan.ThreadID.ValueString()
	userID := plan.UserID.ValueString()

	// PUT add thread member. API usually returns 204 for @me and 204/200 for others.
	if err := r.c.DoJSONWithReason(ctx, "PUT", threadMemberPath(threadID, userID), nil, nil, nil, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", threadID, userID))
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *threadMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state threadMemberResourceModel
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

func (r *threadMemberResource) readIntoState(ctx context.Context, state *threadMemberResourceModel, diags discordFrameworkDiagnostics) {
	threadID := state.ThreadID.ValueString()
	userID := state.UserID.ValueString()
	if threadID == "" || userID == "" {
		tid, uid, err := fwutil.ParseTwoIDs(state.ID.ValueString())
		if err != nil {
			diags.AddError("Invalid ID", "Expected thread_id:user_id")
			return
		}
		threadID, userID = tid, uid
	}

	var out restThreadMemberResource
	if err := r.c.DoJSON(ctx, "GET", threadMemberPath(threadID, userID), nil, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s:%s", threadID, userID))
	state.ThreadID = types.StringValue(threadID)
	state.UserID = types.StringValue(userID)
	state.JoinTimestamp = types.StringValue(out.JoinTimestamp)
	state.Flags = types.Int64Value(int64(out.Flags))
}

func (r *threadMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported operation", "discord_thread_member does not support updates (replace on change)")
}

func (r *threadMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state threadMemberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	threadID := state.ThreadID.ValueString()
	userID := state.UserID.ValueString()

	if err := r.c.DoJSONWithReason(ctx, "DELETE", threadMemberPath(threadID, userID), nil, nil, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *threadMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: thread_id:user_id
	threadID, userID, err := fwutil.ParseTwoIDs(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected thread_id:user_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("thread_id"), threadID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), fmt.Sprintf("%s:%s", threadID, userID))...)
}
