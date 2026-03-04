# arcane_template_variables

Reads global template variables.

## Example Usage

```hcl
data "arcane_template_variables" "vars" {}

output "variables" {
  value = data.arcane_template_variables.vars.variables
}
```

## Argument Reference

This data source has no arguments.

## Attributes Reference

- `variables` (Map of String) - key/value template variables.
- `data_json` (String) - full variable payload as JSON.
