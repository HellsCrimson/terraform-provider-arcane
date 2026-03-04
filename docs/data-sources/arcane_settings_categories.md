# arcane_settings_categories

Lists settings categories.

## Example Usage

```hcl
data "arcane_settings_categories" "all" {}

output "settings_category_count" {
  value = data.arcane_settings_categories.all.total_count
}
```

## Argument Reference

This data source has no arguments.

## Attributes Reference

- `total_count` (Number) - number of categories.
- `data_json` (String) - full category list payload as JSON.
