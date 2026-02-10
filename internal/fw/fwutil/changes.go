package fwutil

import "github.com/hashicorp/terraform-plugin-framework/types"

func ChangedString(plan, state types.String) bool {
	if plan.IsUnknown() || state.IsUnknown() {
		return false
	}
	if plan.IsNull() != state.IsNull() {
		return true
	}
	if plan.IsNull() {
		return false
	}
	return plan.ValueString() != state.ValueString()
}

func ChangedInt64(plan, state types.Int64) bool {
	if plan.IsUnknown() || state.IsUnknown() {
		return false
	}
	if plan.IsNull() != state.IsNull() {
		return true
	}
	if plan.IsNull() {
		return false
	}
	return plan.ValueInt64() != state.ValueInt64()
}

func ChangedBool(plan, state types.Bool) bool {
	if plan.IsUnknown() || state.IsUnknown() {
		return false
	}
	if plan.IsNull() != state.IsNull() {
		return true
	}
	if plan.IsNull() {
		return false
	}
	return plan.ValueBool() != state.ValueBool()
}
