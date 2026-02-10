package fw

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func NewThreadMembersDataSource() datasource.DataSource {
	return &threadMembersDataSource{}
}

type threadMembersDataSource struct {
	c *discord.RestClient
}

type restThreadMember struct {
	ID            string `json:"id"`
	UserID        string `json:"user_id"`
	JoinTimestamp string `json:"join_timestamp"`
	Flags         int    `json:"flags"`
}

type threadMemberModel struct {
	UserID        types.String `tfsdk:"user_id"`
	JoinTimestamp types.String `tfsdk:"join_timestamp"`
	Flags         types.Int64  `tfsdk:"flags"`
}

type threadMembersModel struct {
	ID         types.String        `tfsdk:"id"`
	ThreadID   types.String        `tfsdk:"thread_id"`
	Limit      types.Int64         `tfsdk:"limit"`
	After      types.String        `tfsdk:"after"`
	WithMember types.Bool          `tfsdk:"with_member"`
	Member     []threadMemberModel `tfsdk:"member"`
}

func (d *threadMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_thread_members"
}

func (d *threadMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true},
			"thread_id": schema.StringAttribute{Required: true},
			"limit":     schema.Int64Attribute{Optional: true},
			"after": schema.StringAttribute{
				Optional:    true,
				Description: "Snowflake user ID; return thread members after this user.",
			},
			"with_member": schema.BoolAttribute{
				Optional:    true,
				Description: "Include guild member object where supported.",
			},
			"member": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_id":        schema.StringAttribute{Computed: true},
						"join_timestamp": schema.StringAttribute{Computed: true},
						"flags":          schema.Int64Attribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *threadMembersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	d.c = c.Rest
}

func (d *threadMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data threadMembersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	threadID := data.ThreadID.ValueString()
	q := url.Values{}
	if !data.Limit.IsNull() && !data.Limit.IsUnknown() && data.Limit.ValueInt64() > 0 {
		q.Set("limit", strconv.FormatInt(data.Limit.ValueInt64(), 10))
	}
	if !data.After.IsNull() && data.After.ValueString() != "" {
		q.Set("after", data.After.ValueString())
	}
	if !data.WithMember.IsNull() && !data.WithMember.IsUnknown() {
		if data.WithMember.ValueBool() {
			q.Set("with_member", "true")
		} else {
			q.Set("with_member", "false")
		}
	}

	var out []restThreadMember
	if err := d.c.DoJSON(ctx, "GET", fmt.Sprintf("/channels/%s/thread-members", threadID), q, nil, &out); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	members := make([]threadMemberModel, 0, len(out))
	for _, tm := range out {
		uid := tm.UserID
		if uid == "" {
			uid = tm.ID
		}
		members = append(members, threadMemberModel{
			UserID:        types.StringValue(uid),
			JoinTimestamp: types.StringValue(tm.JoinTimestamp),
			Flags:         types.Int64Value(int64(tm.Flags)),
		})
	}

	data.ID = types.StringValue(threadID)
	data.Member = members
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
