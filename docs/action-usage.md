# Action Details

<!-- == export: inputs / begin == -->

| Name       | Description                                                                                | Required |
| ---------- | ------------------------------------------------------------------------------------------ | :------: |
| `token`    | `GITHUB_TOKEN` or Personal Access Token with `repo` scope                                  |   Yes    |
| `job`      | Target job to check against. If none is provided, it will fall back to check all the jobs. |          |
| `interval` | Check interval for merge-gatekeeper to recheck the job status. Default is set to 30 (sec). |          |
| `timeout`  | Timeout setup for merge-gatekeeper to give up further check. Default is set to 600 (sec).  |          |
| `ref`      | Git ref to check out. This falls back to the HEAD for given PR, but can be set to any ref. |          |

<!-- == export: inputs / end == -->
