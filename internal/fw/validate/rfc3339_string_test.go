package validate

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestRFC3339TimestampValidator(t *testing.T) {
	cases := []struct {
		name    string
		val     types.String
		wantErr bool
	}{
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"empty", types.StringValue(""), false},
		{"ok", types.StringValue("2026-02-11T00:00:00Z"), false},
		{"bad", types.StringValue("2026-02-11"), true},
	}

	v := RFC3339Timestamp()
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
