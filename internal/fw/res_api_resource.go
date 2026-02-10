package fw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
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

func NewAPIResourceResource() resource.Resource {
	return &apiResourceResource{}
}

type apiResourceResource struct {
	c *discord.RestClient
}

type apiResourceModel struct {
	ID types.String `tfsdk:"id"`

	IDField    types.String `tfsdk:"id_field"`
	IDOverride types.String `tfsdk:"id_override"`

	CreateMethod   types.String `tfsdk:"create_method"`
	CreatePath     types.String `tfsdk:"create_path"`
	CreateBodyJSON types.String `tfsdk:"create_body_json"`

	ReadPath      types.String `tfsdk:"read_path"`
	ReadQueryJSON types.String `tfsdk:"read_query_json"`

	UpdateMethod   types.String `tfsdk:"update_method"`
	UpdatePath     types.String `tfsdk:"update_path"`
	UpdateBodyJSON types.String `tfsdk:"update_body_json"`

	DeleteMethod   types.String `tfsdk:"delete_method"`
	DeletePath     types.String `tfsdk:"delete_path"`
	DeleteBodyJSON types.String `tfsdk:"delete_body_json"`

	Reason types.String `tfsdk:"reason"`

	ResponseJSON types.String `tfsdk:"response_json"`
}

func (r *apiResourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_resource"
}

func (r *apiResourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	methodValidator := []validator.String{
		validate.OneOf("SKIP", "GET", "POST", "PUT", "PATCH", "DELETE"),
	}

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},

			"id_field": schema.StringAttribute{
				Optional:    true,
				Description: "Field name to extract as resource ID from create response (JSON object).",
			},
			"id_override": schema.StringAttribute{
				Optional:    true,
				Description: "If set, uses this value as the Terraform resource ID instead of parsing the create response.",
			},

			"create_method": schema.StringAttribute{
				Optional:   true,
				Validators: methodValidator,
			},
			"create_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path for create call. May include '{id}' placeholder.",
			},
			"create_body_json": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					validate.JSONString(),
				},
				PlanModifiers: []planmodifier.String{
					planmod.NormalizeJSONString(),
				},
			},

			"read_path": schema.StringAttribute{
				Required:    true,
				Description: "Path for read call. Should include '{id}' placeholder.",
			},
			"read_query_json": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					validate.JSONString(),
				},
				PlanModifiers: []planmodifier.String{
					planmod.NormalizeJSONString(),
				},
				Description: "JSON object of query parameters for read.",
			},

			"update_method": schema.StringAttribute{
				Optional:   true,
				Validators: methodValidator,
			},
			"update_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path for update call. Defaults to read_path.",
			},
			"update_body_json": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					validate.JSONString(),
				},
				PlanModifiers: []planmodifier.String{
					planmod.NormalizeJSONString(),
				},
			},

			"delete_method": schema.StringAttribute{
				Optional:   true,
				Validators: methodValidator,
			},
			"delete_path": schema.StringAttribute{
				Optional:    true,
				Description: "Path for delete call. Defaults to read_path.",
			},
			"delete_body_json": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Validators: []validator.String{
					validate.JSONString(),
				},
				PlanModifiers: []planmodifier.String{
					planmod.NormalizeJSONString(),
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

			"response_json": schema.StringAttribute{
				Computed:    true,
				Description: "Normalized JSON response from read.",
			},
		},
	}
}

func (r *apiResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, diags := getContextFromProviderData(req.ProviderData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.c = c.Rest
}

func (r *apiResourceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Replicates the old CustomizeDiff logic for create inputs.
	var plan apiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	method := strings.ToUpper(strings.TrimSpace(plan.CreateMethod.ValueString()))
	if method == "" {
		method = "POST"
	}
	createPath := strings.TrimSpace(plan.CreatePath.ValueString())
	idOverride := strings.TrimSpace(plan.IDOverride.ValueString())

	if method == "SKIP" {
		if idOverride == "" {
			resp.Diagnostics.AddAttributeError(path.Root("id_override"), "Invalid configuration", "id_override is required when create_method=SKIP")
		}
		return
	}

	if createPath == "" {
		resp.Diagnostics.AddAttributeError(path.Root("create_path"), "Invalid configuration", "create_path is required when create_method is not SKIP")
		return
	}
	if strings.Contains(createPath, "{id}") && idOverride == "" {
		resp.Diagnostics.AddAttributeError(path.Root("id_override"), "Invalid configuration", "id_override is required when create_path contains {id}")
		return
	}
}

func (r *apiResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applyDefaults(&plan)

	method := strings.ToUpper(plan.CreateMethod.ValueString())
	idOverride := strings.TrimSpace(plan.IDOverride.ValueString())

	if method == "SKIP" {
		plan.ID = types.StringValue(idOverride)
		r.readIntoState(ctx, &plan, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	path := substituteID(plan.CreatePath.ValueString(), idOverride)
	body, err := parseJSONBody(plan.CreateBodyJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid create_body_json", err.Error())
		return
	}

	var out any
	if err := r.c.DoJSONWithReason(ctx, method, path, nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	if idOverride != "" {
		plan.ID = types.StringValue(idOverride)
		r.readIntoState(ctx, &plan, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	idField := plan.IDField.ValueString()
	if idField == "" {
		idField = "id"
	}

	if mm, ok := out.(map[string]any); ok {
		if idv, ok := mm[idField]; ok {
			if ids, ok := idv.(string); ok && ids != "" {
				plan.ID = types.StringValue(ids)
				r.readIntoState(ctx, &plan, &resp.Diagnostics)
				resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
				return
			}
		}
	}

	// Fallback: use hash of create path + normalized response.
	b, _ := json.Marshal(out)
	norm, _ := discord.NormalizeJSON(string(b))
	plan.ID = types.StringValue(fmt.Sprintf("%d", discord.Hashcode(path+"|"+norm)))
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiResourceModel
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

func (r *apiResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiResourceModel
	var state apiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applyDefaults(&plan)

	method := strings.ToUpper(plan.UpdateMethod.ValueString())
	if method == "SKIP" {
		plan.ID = state.ID
		r.readIntoState(ctx, &plan, &resp.Diagnostics)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		return
	}

	path := plan.UpdatePath.ValueString()
	if strings.TrimSpace(path) == "" {
		path = plan.ReadPath.ValueString()
	}
	path = substituteID(path, state.ID.ValueString())

	body, err := parseJSONBody(plan.UpdateBodyJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid update_body_json", err.Error())
		return
	}

	var out any
	if err := r.c.DoJSONWithReason(ctx, method, path, nil, body, &out, plan.Reason.ValueString()); err != nil {
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}

	plan.ID = state.ID
	r.readIntoState(ctx, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applyDefaults(&state)

	method := strings.ToUpper(state.DeleteMethod.ValueString())
	if method == "SKIP" {
		resp.State.RemoveResource(ctx)
		return
	}

	path := state.DeletePath.ValueString()
	if strings.TrimSpace(path) == "" {
		path = state.ReadPath.ValueString()
	}
	path = substituteID(path, state.ID.ValueString())

	body, err := parseJSONBody(state.DeleteBodyJSON.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid delete_body_json", err.Error())
		return
	}

	if err := r.c.DoJSONWithReason(ctx, method, path, nil, body, nil, state.Reason.ValueString()); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Discord API error", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *apiResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func applyDefaults(m *apiResourceModel) {
	if m.IDField.IsNull() || m.IDField.ValueString() == "" {
		m.IDField = types.StringValue("id")
	}
	if m.CreateMethod.IsNull() || strings.TrimSpace(m.CreateMethod.ValueString()) == "" {
		m.CreateMethod = types.StringValue("POST")
	}
	if m.UpdateMethod.IsNull() || strings.TrimSpace(m.UpdateMethod.ValueString()) == "" {
		m.UpdateMethod = types.StringValue("PATCH")
	}
	if m.DeleteMethod.IsNull() || strings.TrimSpace(m.DeleteMethod.ValueString()) == "" {
		m.DeleteMethod = types.StringValue("DELETE")
	}
}

func substituteID(p, id string) string {
	return strings.ReplaceAll(p, "{id}", id)
}

func parseJSONBody(s string) (any, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, err
	}
	return v, nil
}

func parseQueryJSON(s string) (url.Values, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	q := url.Values{}
	for k, v := range m {
		switch t := v.(type) {
		case string:
			q.Set(k, t)
		case bool:
			if t {
				q.Set(k, "true")
			} else {
				q.Set(k, "false")
			}
		default:
			b, _ := json.Marshal(t)
			q.Set(k, string(b))
		}
	}
	return q, nil
}

func (r *apiResourceResource) readIntoState(ctx context.Context, state *apiResourceModel, diags discordFrameworkDiagnostics) {
	path := substituteID(state.ReadPath.ValueString(), state.ID.ValueString())
	q, err := parseQueryJSON(state.ReadQueryJSON.ValueString())
	if err != nil {
		diags.AddError("Invalid read_query_json", err.Error())
		return
	}

	var out any
	if err := r.c.DoJSON(ctx, "GET", path, q, nil, &out); err != nil {
		if discord.IsDiscordHTTPStatus(err, 404) {
			state.ID = types.StringNull()
			return
		}
		diags.AddError("Discord API error", err.Error())
		return
	}

	b, err := json.Marshal(out)
	if err != nil {
		diags.AddError("JSON error", err.Error())
		return
	}
	norm, err := discord.NormalizeJSON(string(b))
	if err != nil {
		diags.AddError("JSON error", err.Error())
		return
	}

	state.ResponseJSON = types.StringValue(norm)
}
