# arcane_template

Manages a reusable docker-compose template.

## Example Usage

```hcl
resource "arcane_template" "nginx" {
  name        = "nginx-proxy"
  description = "Nginx reverse proxy template"
  content     = <<-EOT
    version: '3'
    services:
      nginx:
        image: nginx:latest
        ports:
          - "${NGINX_PORT}:80"
        volumes:
          - ./nginx.conf:/etc/nginx/nginx.conf:ro
  EOT
  env_content = <<-EOT
    NGINX_PORT=8080
  EOT
}
```

## Argument Reference

### Required

- `name` (String) - Name of the template.
- `description` (String) - Description of the template.
- `content` (String) - Docker Compose YAML content.
- `env_content` (String) - Environment variables template content (.env format).

## Attributes Reference

- `id` (String) - Unique identifier of the template.
- `is_custom` (Boolean) - Whether this is a custom template.
- `is_remote` (Boolean) - Whether this template is from a remote registry.
- `registry_id` (String) - ID of the registry this template belongs to (if remote).

## Import

Import using the template ID:

```
terraform import arcane_template.nginx <template_id>
```
