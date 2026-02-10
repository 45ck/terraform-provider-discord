package planmod

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// TrimTrailingCRLFString trims a single trailing "\r\n" from a string value during planning.
// This matches historical DiffSuppress behavior for message content.
func TrimTrailingCRLFString() planmodifier.String {
	return trimTrailingCRLFString{}
}

type trimTrailingCRLFString struct{}

func (m trimTrailingCRLFString) Description(_ context.Context) string {
	return "Trim a single trailing CRLF for diff stability."
}

func (m trimTrailingCRLFString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m trimTrailingCRLFString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	s := req.ConfigValue.ValueString()
	resp.PlanValue = types.StringValue(strings.TrimSuffix(s, "\r\n"))
}
