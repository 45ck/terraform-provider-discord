package validate

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestSnowflakeValidator(t *testing.T) {
	cases := []struct {
		name    string
		val     types.String
		wantErr bool
	}{
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"empty", types.StringValue(""), false},
		{"ok17", types.StringValue("12345678901234567"), false},
		{"ok20", types.StringValue("12345678901234567890"), false},
		{"short", types.StringValue("123"), true},
		{"nondigit", types.StringValue("abc1234567890123456"), true},
	}

	v := Snowflake()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("x"),
				ConfigValue: tc.val,
			}
			resp := validator.StringResponse{Diagnostics: diag.Diagnostics{}}
			v.ValidateString(context.Background(), req, &resp)
			if tc.wantErr && !resp.Diagnostics.HasError() {
				t.Fatalf("expected error, got none")
			}
			if !tc.wantErr && resp.Diagnostics.HasError() {
				t.Fatalf("expected no error, got: %v", resp.Diagnostics)
			}
		})
	}
}

func TestSnowflakeOrAtMeValidator(t *testing.T) {
	cases := []struct {
		name    string
		val     types.String
		wantErr bool
	}{
		{"okAtMe", types.StringValue("@me"), false},
		{"okSnowflake", types.StringValue("12345678901234567"), false},
		{"bad", types.StringValue("@you"), true},
	}

	v := SnowflakeOrAtMe()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("x"),
				ConfigValue: tc.val,
			}
			resp := validator.StringResponse{Diagnostics: diag.Diagnostics{}}
			v.ValidateString(context.Background(), req, &resp)
			if tc.wantErr && !resp.Diagnostics.HasError() {
				t.Fatalf("expected error, got none")
			}
			if !tc.wantErr && resp.Diagnostics.HasError() {
				t.Fatalf("expected no error, got: %v", resp.Diagnostics)
			}
		})
	}
}
