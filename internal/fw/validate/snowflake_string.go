package validate

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var snowflakeRe = regexp.MustCompile(`^[0-9]{17,20}$`)

// Snowflake validates that a string looks like a Discord snowflake (17-20 digits).
// It is a syntactic check only.
func Snowflake() validator.String {
	return snowflakeStringValidator{}
}

// SnowflakeOrAtMe validates a snowflake or "@me".
func SnowflakeOrAtMe() validator.String {
	return snowflakeOrAtMeStringValidator{}
}

type snowflakeStringValidator struct{}

func (v snowflakeStringValidator) Description(_ context.Context) string {
	return "Value must be a Discord snowflake ID."
}

func (v snowflakeStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v snowflakeStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	raw := strings.TrimSpace(req.ConfigValue.ValueString())
	if raw == "" {
		return
	}

	if snowflakeRe.MatchString(raw) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid ID",
		fmt.Sprintf("Expected a Discord snowflake (17-20 digits), got %q", raw),
	)
}

type snowflakeOrAtMeStringValidator struct{}

func (v snowflakeOrAtMeStringValidator) Description(_ context.Context) string {
	return "Value must be a Discord snowflake ID or @me."
}

func (v snowflakeOrAtMeStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v snowflakeOrAtMeStringValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	raw := strings.TrimSpace(req.ConfigValue.ValueString())
	if raw == "" {
		return
	}

	if raw == "@me" || snowflakeRe.MatchString(raw) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Invalid ID",
		fmt.Sprintf("Expected a Discord snowflake (17-20 digits) or \"@me\", got %q", raw),
	)
}
