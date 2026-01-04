# Arcane Provider

The Arcane provider allows managing Arcane via its HTTP API using an API key.

## Example Usage

```
provider "arcane" {
  api_key  = var.arcane_api_key
  endpoint = "http://localhost:3552/api"
}
```

## Argument Reference

- `api_key` (String, Sensitive) — API key; alternatively set `ARCANE_API_KEY`.
- `endpoint` (String) — Base API URL. Defaults to `http://localhost:3552/api`.

## Authentication

Uses header `X-API-Key` per the OpenAPI spec.

