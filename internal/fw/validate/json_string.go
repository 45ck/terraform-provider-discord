package validate

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// JSONString validates that a string is syntactically valid JSON.
// It does not enforce the top-level type (object/array/etc.).
func JSONString() validator.String {
	return jsonStringValidator{}
}

type jsonStringValidator struct{}

func (v jsonStringValidator) Description(_ context.Context) string {
	return "Value must be valid JSON."
}

func (v jsonStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v jsonStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	raw := req.ConfigValue.ValueString()
	if raw == "" {
		return
	}

	var tmp any
	if err := json.Unmarshal([]byte(raw), &tmp); err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid JSON",
			err.Error(),
		)
		return
	}
}
