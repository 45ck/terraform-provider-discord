package planmod

import (
	"context"

	"github.com/aequasi/discord-terraform/discord"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NormalizeJSONString normalizes JSON string values during planning so semantically equal JSON
// does not cause perpetual diffs due to whitespace/key ordering.
func NormalizeJSONString() planmodifier.String {
	return normalizeJSONString{}
}

type normalizeJSONString struct{}

func (m normalizeJSONString) Description(_ context.Context) string {
	return "Normalize JSON to a stable encoding for diff stability."
}

func (m normalizeJSONString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m normalizeJSONString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	raw := req.ConfigValue.ValueString()
	norm, err := discord.NormalizeJSON(raw)
	if err != nil {
		// Leave as-is; schema validation should surface JSON errors where required.
		return
	}
	resp.PlanValue = types.StringValue(norm)
}
