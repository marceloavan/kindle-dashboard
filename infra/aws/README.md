# AWS scheduler (reliable clock)

GitHub's own cron scheduler delivered only ~3 runs/day on this repo, so the
render clock lives in AWS: an EventBridge Scheduler fires every 15 minutes
during daytime hours and posts to the GitHub REST API, dispatching the
`render.yml` workflow (which then renders and deploys to Pages as usual).

## How it's applied

This root follows the **workhours-manager pattern** — no local `terraform
apply`. Plans run automatically on pull requests; applies run via a `/apply`
comment, authenticated by GitHub OIDC:

- **Roles are REUSED from the workhours-manager repo** (its `base/oidc.tf`
  grants `marceloavan/kindle-dashboard` subjects to the existing
  `whours-github-actions-terraform-plan` / `-apply` roles). No roles live in
  this repo.
- Plan workflow: `.github/workflows/terraform-aws.yml`
  (assumes the plan role; uses a placeholder for the token).
- Apply workflow: `.github/workflows/terraform-aws-apply.yml`
  (gated by the `aws-prd` environment; reads the real token from the
  `GH_DISPATCH_PAT` environment secret).

## One-time setup (already done / to do)

1. workhours-manager PR granting the OIDC reuse — **must be applied first**.
2. `aws-prd` environment exists in this repo.
3. Set the secret (fine-grained PAT, only this repo, **Actions: Read and
   write**):

```sh
gh secret set GH_DISPATCH_PAT --env aws-prd --repo marceloavan/kindle-dashboard
```

## Applying changes

```text
/apply   # as a comment on the PR
```

## Notes

- Schedule: `cron(*/15 7-22 * * *)` in `America/Sao_Paulo` (07:00–22:45 BRT).
- The GitHub Actions `schedule` trigger stays as a free backup.
- Rotate the token before expiry: update the secret, then `/apply` again.
