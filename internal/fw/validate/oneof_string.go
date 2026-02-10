package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// OneOf validates that the value (uppercased) is one of the given strings.
func OneOf(values ...string) validator.String {
	return oneOfStringValidator{values: values}
}

type oneOfStringValidator struct {
	values []string
}

func (v oneOfStringValidator) Description(_ context.Context) string {
	return "Value must be one of the allowed values."
}

func (v oneOfStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v oneOfStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	raw := strings.ToUpper(strings.TrimSpace(req.ConfigValue.ValueString()))
	if raw == "" {
		return
	}

	for _, allowed := range v.values {
		if raw == allowed {
			return
		}
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid value",
		fmt.Sprintf("Value must be one of: %s", strings.Join(v.values, ", ")),
	)
}
