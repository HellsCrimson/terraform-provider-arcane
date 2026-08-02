package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// serverComputedAttr names a computed attribute Update rewrites from the API
// response, together with the typed unknown to plan for it.
type serverComputedAttr struct {
	name    string
	unknown attr.Value
}

// planServerComputedUnknown marks the attributes Update rewrites from the API
// response as "known after apply", so the plan stops promising values only the
// server gets to decide.
//
// Terraform fills computed attributes in the plan from the prior state and lets
// the framework turn them into unknowns only when the *configuration* already
// changed something: MarkComputedNilsAsUnknown is skipped when the proposed plan
// still equals the prior state. A plan that exists solely because ModifyPlan
// produced it therefore carries the prior status, counts and so on straight into
// an update that overwrites them, and Terraform rejects the result:
//
//	Error: Provider produced inconsistent result after apply
//	... produced an unexpected new value: .status: was cty.StringVal("stopped"),
//	but now cty.StringVal("running").
//
// Both resources here have exactly such plans: arcane_project_path picks up
// compose/env file changes the configuration never mentions, and
// redeploy_trigger = "always" plans an update through last_redeploy alone. A
// redeploy then restarts the project, which moves status during the apply
// itself — no refresh beforehand could have predicted it.
//
// Nothing is marked when no update is planned, so empty plans stay empty.
func planServerComputedUnknown(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, attrs []serverComputedAttr) {
	if req.State.Raw.IsNull() || resp.Plan.Raw.IsNull() {
		// Create plans everything unknown already; destroy plans nothing.
		return
	}
	if resp.Plan.Raw.Equal(req.State.Raw) {
		// No change planned, so Terraform will not call Update and nothing can
		// overwrite these values.
		return
	}

	for _, a := range attrs {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root(a.name), a.unknown)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}
