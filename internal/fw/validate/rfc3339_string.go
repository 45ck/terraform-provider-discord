package validate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// RFC3339Timestamp validates that a string is a RFC3339 timestamp.
// Empty string is allowed (common for optional timestamps).
func RFC3339Timestamp() validator.String {
	return rfc3339TimestampStringValidator{}
}

type rfc3339TimestampStringValidator struct{}

func (v rfc3339TimestampStringValidator) Description(_ context.Context) string {
	return "Value must be an RFC3339 timestamp."
}

func (v rfc3339TimestampStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v rfc3339TimestampStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	raw := strings.TrimSpace(req.ConfigValue.ValueString())
	if raw == "" {
		return
	}

	if _, err := time.Parse(time.RFC3339, raw); err == nil {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid timestamp",
		fmt.Sprintf("Expected RFC3339 timestamp, got %q", raw),
	)
}
