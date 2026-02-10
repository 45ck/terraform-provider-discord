package planmod

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// IgnoreChangesString forces the planned value to remain whatever is in prior state.
// This is useful for write-only fields (e.g. audit log reason) that are not readable.
func IgnoreChangesString() planmodifier.String {
	return ignoreChangesString{}
}

type ignoreChangesString struct{}

func (m ignoreChangesString) Description(_ context.Context) string {
	return "Ignore changes to this attribute; keep prior state value."
}

func (m ignoreChangesString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m ignoreChangesString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	// If no prior state exists, allow the config value through.
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		return
	}
	// Only override when config is set/known; unknowns should remain unknown.
	if req.ConfigValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}
