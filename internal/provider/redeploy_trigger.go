package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Redeploy trigger modes shared by arcane_project and arcane_project_path.
//
//   - never:   never redeploy, whatever changes.
//   - default: redeploy when the compose/env content changed (the historical
//     meaning of redeploy_on_update = true).
//   - update:  redeploy whenever the resource is updated in place, even when the
//     change does not touch the compose/env content.
//   - always:  redeploy on every apply. Terraform only calls Update when the plan
//     is non-empty, so this mode makes the provider mark last_redeploy as unknown
//     during plan, which means the resource always reports a change.
const (
	redeployTriggerNever   = "never"
	redeployTriggerDefault = "default"
	redeployTriggerUpdate  = "update"
	redeployTriggerAlways  = "always"
)

var redeployTriggerValues = []string{
	redeployTriggerNever,
	redeployTriggerDefault,
	redeployTriggerUpdate,
	redeployTriggerAlways,
}

const redeployTriggerDescription = "When to redeploy the project: `never` never redeploys; `default` redeploys when the compose/env content changed; `update` redeploys on any in-place update of this resource; `always` redeploys on every apply. `always` makes the resource report a change on every plan (`last_redeploy` becomes \"known after apply\"), which is the only way Terraform will call the provider when nothing else changed. Defaults to `default`."

const lastRedeployDescription = "Timestamp (RFC3339) of the last redeploy performed by the provider. Null until the provider has redeployed the project at least once."

// resolveRedeployTrigger returns the effective redeploy trigger for a plan,
// reading the *configuration* rather than the plan so an unset attribute can be
// told apart from one explicitly set to its default value.
//
// legacyBoolAttr is the name of the deprecated boolean attribute that used to
// carry this behaviour (redeploy_on_update); pass an empty string for resources
// that never had one. It is only consulted when redeploy_trigger is absent from
// the configuration, so the new attribute always wins.
func resolveRedeployTrigger(ctx context.Context, config tfsdk.Config, legacyBoolAttr string) (string, diag.Diagnostics) {
	var diags diag.Diagnostics

	var trigger types.String
	diags.Append(config.GetAttribute(ctx, path.Root("redeploy_trigger"), &trigger)...)
	if diags.HasError() {
		return redeployTriggerDefault, diags
	}
	if !trigger.IsNull() && !trigger.IsUnknown() {
		return trigger.ValueString(), diags
	}

	if legacyBoolAttr != "" {
		var legacy types.Bool
		diags.Append(config.GetAttribute(ctx, path.Root(legacyBoolAttr), &legacy)...)
		if diags.HasError() {
			return redeployTriggerDefault, diags
		}
		if !legacy.IsNull() && !legacy.IsUnknown() && !legacy.ValueBool() {
			return redeployTriggerNever, diags
		}
	}

	return redeployTriggerDefault, diags
}

// resolvedRedeployAttrs returns the redeploy attribute values to store, filling
// in any the plan left unknown because its configuration could not be evaluated
// during plan (a value derived from another resource, say). Configuration is
// always known by the time Create/Update run.
//
// Resources without a deprecated boolean pass a null planLegacy and ignore the
// second return value.
func resolvedRedeployAttrs(ctx context.Context, config tfsdk.Config, legacyBoolAttr string, planTrigger types.String, planLegacy types.Bool) (types.String, types.Bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !planTrigger.IsUnknown() && !planLegacy.IsUnknown() {
		return planTrigger, planLegacy, diags
	}

	resolved, d := resolveRedeployTrigger(ctx, config, legacyBoolAttr)
	diags.Append(d...)
	if diags.HasError() {
		return planTrigger, planLegacy, diags
	}
	if planTrigger.IsUnknown() {
		planTrigger = types.StringValue(resolved)
	}
	if planLegacy.IsUnknown() {
		planLegacy = types.BoolValue(resolved != redeployTriggerNever)
	}
	return planTrigger, planLegacy, diags
}

// shouldRedeploy reports whether an in-place update must redeploy the project.
// It is only ever called from Update, which Terraform invokes only when the plan
// is non-empty, so "update" and "always" both mean yes by the time we get here.
func shouldRedeploy(trigger string, contentChanged bool) bool {
	switch trigger {
	case redeployTriggerNever:
		return false
	case redeployTriggerUpdate, redeployTriggerAlways:
		return true
	default:
		return contentChanged
	}
}

// redeployPlanned mirrors shouldRedeploy at plan time, where "update" means
// "the plan is not empty" and "always" is unconditional.
func redeployPlanned(trigger string, contentChanged, planHasChanges bool) bool {
	switch trigger {
	case redeployTriggerNever:
		return false
	case redeployTriggerAlways:
		return true
	case redeployTriggerUpdate:
		return planHasChanges
	default:
		return contentChanged
	}
}

// redeployTimestamp is the value stored in last_redeploy after a redeploy.
func redeployTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}
