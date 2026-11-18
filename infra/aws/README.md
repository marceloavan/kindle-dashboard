# AWS scheduler (reliable clock)

GitHub's own cron scheduler delivered only ~3 runs/day on this repo, so the
render clock lives in AWS: an EventBridge Scheduler fires every 15 minutes
during daytime hours and posts to the GitHub REST API, dispatching the
`render.yml` workflow (which then renders and deploys to Pages as usual).

## Setup

1. Create a **fine-grained personal access token** on GitHub:
   - Settings → Developer settings → Personal access tokens → Fine-grained tokens
   - Repository access: only `marceloavan/kindle-dashboard`
   - Permissions: Actions → **Read and write**
2. Apply (token passed via env var, never committed):

```sh
cd infra/aws
terraform init
TF_VAR_github_token=<token> terraform plan
TF_VAR_github_token=<token> terraform apply
```

## Notes

- Schedule: `cron(*/15 7-22 * * *)` in `America/Sao_Paulo` (07:00–22:45 BRT).
- The GitHub Actions `schedule` trigger stays as a free backup.
- Rotate the token before its expiry date: re-run `terraform apply` with the
  new value.
