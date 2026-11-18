package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Secrets are write-only: "<name>_wo" is sent to the provider but never
// persisted. The framework nulls it in the plan and in every state it hands
// back to Terraform, so its value has to be read from the configuration
// (req.Config). Terraform cannot diff a value it does not keep, so it is paired
// with "<name>_wo_version": changing that number is what makes an Update send
// the new value.

// writeOnlyAttribute is the write-only secret `<name>_wo`. It requires its
// version attribute.
func writeOnlyAttribute(name, description string, validators ...validator.String) resourceschema.StringAttribute {
	return resourceschema.StringAttribute{
		Optional:  true,
		Sensitive: true,
		WriteOnly: true,
		Description: fmt.Sprintf("%s Write-only: never stored in state or plan files (requires Terraform or OpenTofu 1.11+). "+
			"Changes are only sent when `%s_wo_version` changes.", description, name),
		Validators: append([]validator.String{
			stringvalidator.AlsoRequires(path.MatchRoot(name + "_wo_version")),
		}, validators...),
	}
}

// writeOnlyVersionAttribute is the version companion of `<name>_wo`.
func writeOnlyVersionAttribute(name string, modifiers ...planmodifier.Int64) resourceschema.Int64Attribute {
	return resourceschema.Int64Attribute{
		Optional:      true,
		Description:   fmt.Sprintf("Version of `%[1]s_wo`. Terraform does not keep write-only values, so change this number to send a new `%[1]s_wo` to Arcane.", name),
		PlanModifiers: modifiers,
	}
}

// writeOnlySend reports whether an Update must send the configured write-only
// value: only when its version changed, and only when a value is configured.
func writeOnlySend(configured types.String, plannedVersion, priorVersion types.Int64) bool {
	return !plannedVersion.Equal(priorVersion) && isSetString(configured)
}

// isSetString reports whether a string holds a non-empty, known value.
func isSetString(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
}
