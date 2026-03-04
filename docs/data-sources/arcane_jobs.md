# arcane_jobs

Lists background jobs for an environment.

## Example Usage

```hcl
data "arcane_jobs" "env_jobs" {
  environment_id = "env-123456"
}

output "job_count" {
  value = data.arcane_jobs.env_jobs.count
}
```

## Argument Reference

- `environment_id` (String, Required) - environment ID.

## Attributes Reference

- `is_agent` (Bool) - whether the environment is agent-based.
- `count` (Number) - number of jobs returned.
- `jobs_json` (String) - full jobs payload as JSON.
