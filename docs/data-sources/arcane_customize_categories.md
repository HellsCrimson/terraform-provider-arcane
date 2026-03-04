# arcane_customize_categories

Lists customization categories.

## Example Usage

```hcl
data "arcane_customize_categories" "all" {}

output "category_count" {
  value = data.arcane_customize_categories.all.count
}
```

## Argument Reference

This data source has no arguments.

## Attributes Reference

- `count` (Number) - number of categories.
- `data_json` (String) - full category list payload as JSON.
