# Action Details

## Action Inputs

<!-- == export: inputs / begin == -->

| Name       | Description                                                                                                                                                                                                                                                                                          | Required |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------: |
| `token`    | `GITHUB_TOKEN` or Personal Access Token with `repo` scope                                                                                                                                                                                                                                            |   Yes    |
| `self`     | The name of Merge Gatekeeper job, and defaults to `merge-gatekeeper`. This is used to check other job status, and do not check Merge Gatekeeper itself. If you updated the GitHub Action job name from `merge-gatekeeper` to something else, you would need to specify the new name with this value. |          |
| `interval` | Check interval to recheck the job status. Default is set to 5 (sec). Accepts seconds (`30`) or a duration (`30s`, `1m`). Must be at least 1 second.                                                                                                                                                 |          |
| `timeout`  | Timeout setup to give up further check. Default is set to 600 (sec). Accepts seconds (`600`) or a duration (`10m`, `1h30m`).                                                                                                                                                                          |          |
| `initial-delay` | Delay before the first validation. Useful when other jobs are known to take a while, to avoid needless polling at the start. This delay is not counted towards `timeout`. Default is set to 0 (sec). Accepts seconds or a duration (`1m`).                                                     |          |
| `ignored`  | Jobs to ignore regardless of their statuses. Defined as a comma-separated list.                                                                                                                                                                                                                      |          |
| `ref`      | Git ref to check. This defaults to the HEAD for given PR. When that is empty, such as on `push` events, it falls back to the commit that triggered the workflow (`GITHUB_SHA`). Can be set to any ref.                                                                                                 |          |

<!-- == export: inputs / end == -->

## Runner Requirements

<!-- == export: runner / begin == -->

Merge Gatekeeper is a JavaScript action that runs on the `node24` runtime bundled with the GitHub Actions runner. It does not need Docker, and it does not need `actions/checkout` or `actions/setup-node`.

- **We recommend `runs-on: ubuntu-slim`.** The job mostly waits for other jobs and only makes a few API calls, so the [single-CPU Linux runner](https://docs.github.com/en/actions/reference/runners/github-hosted-runners#single-cpu-runners) (1 vCPU, 5 GB RAM, runs in a container) is enough. In private repositories it is billed at a third of the standard Linux runner ([pricing](https://docs.github.com/en/billing/reference/actions-runner-pricing)); in public repositories both are free. v1, a Docker container action, was not a fit for it: `ubuntu-slim` runs in an unprivileged container without Docker-in-Docker support.
- **`ubuntu-slim` jobs are stopped after 15 minutes.** Keep `initial-delay` plus `timeout` well below that; the defaults (no delay, `timeout: 600`) fit. If your other jobs take longer than about 14 minutes, use `runs-on: ubuntu-latest` instead, which has the usual 6-hour job limit.
- macOS and Windows runners also work, but they are billed at a higher rate for the same wait.
- **Self-hosted runners** must run a version of the Actions runner that supports `node24` actions. Docker is not required, so runners inside containers or without Docker access work as well.
- **GitHub Enterprise Server** is supported. The API endpoint is taken from `GITHUB_API_URL`, which the runner sets automatically.
- **Network access** to the GitHub REST API (`api.github.com`, or your Enterprise Server API) is the only outbound connection Merge Gatekeeper makes. If the runner reaches it through a proxy, `HTTPS_PROXY` / `HTTP_PROXY` and `NO_PROXY` (upper or lower case) are honoured. This needs the `node24` runtime of Actions runner with Node.js 24.14 or later; on older runners, set `NODE_USE_ENV_PROXY: 1` in the job `env`.

<!-- == export: runner / end == -->

## Token Permissions

<!-- == export: permissions / begin == -->

| Permission       | Needed for                                                                                                                                   |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `checks: read`   | Required. Reads check runs, which include GitHub Actions jobs.                                                                               |
| `statuses: read` | Required. Reads commit statuses.                                                                                                             |
| `actions: read`  | Recommended. Tells jobs with the same name in different workflows apart ([#60](https://github.com/upsidr/merge-gatekeeper/issues/60)). |

Without `actions: read`, a private repository cannot read which workflow a job belongs to. Merge Gatekeeper then shows a warning and treats jobs with the same name as one job, where only the latest run counts. Public repositories can read it without the permission. See [how it works](/docs/details.md#how-does-merge-gatekeeper-work) for details.

<!-- == export: permissions / end == -->

## Upgrading from v1

<!-- == export: upgrade / begin == -->

Change `upsidr/merge-gatekeeper@v1` to `@v2`. The inputs and their defaults are unchanged.

What changes in v2:

- It runs as a JavaScript action, without Docker. There is no image to pull and no build step. See [why](/docs/details.md#why-is-merge-gatekeeper-a-javascript-action).
- Commits with 100 or more commit statuses no longer cause a timeout ([#98](https://github.com/upsidr/merge-gatekeeper/issues/98)).
- A completed check run that has no conclusion yet is treated as still running, instead of crashing.
- GitHub Enterprise Server is supported through `GITHUB_API_URL` ([#74](https://github.com/upsidr/merge-gatekeeper/issues/74)).

If you use a self-hosted runner, check that it supports `node24` actions before upgrading.

<!-- == export: upgrade / end == -->

## Usage

### Copy Standard YAML

<!-- == export: simple-usage / begin == -->

The easiest approach is to copy the standard definition, and save it under `.github/workflows` directory. There is no further modification required unless you have some specific requirements.

#### With `curl`

```bash
curl -sSL https://raw.githubusercontent.com/upsidr/merge-gatekeeper/main/example/merge-gatekeeper.yml \
  > .github/workflows/merge-gatekeeper.yml
```

#### Directly copy YAML

The below is the copy of [`/example/merge-gatekeeper.yml`](/example/merge-gatekeeper.yml), with extra comments.

<!-- == imptr: basic-yaml / begin from: ../example/definitions.yaml#[standard-setup] wrap: yaml == -->
```yaml
---
name: Merge Gatekeeper

on:
  pull_request:
    branches:
      - main
      - master

jobs:
  merge-gatekeeper:
    # The cheapest runner is enough: the job mostly waits. ubuntu-slim jobs are
    # stopped after 15 minutes, so use ubuntu-latest if you raise `timeout`
    # (default 10 minutes) close to that.
    runs-on: ubuntu-slim
    # Restrict permissions of the GITHUB_TOKEN.
    # Docs: https://docs.github.com/en/actions/using-jobs/assigning-permissions-to-jobs
    permissions:
      actions: read # Tells jobs with the same name in different workflows apart.
      checks: read
      statuses: read
    steps:
      - name: Run Merge Gatekeeper
        # NOTE: For supply-chain safety, pin to a commit SHA and keep the version
        #       tag in a trailing comment, e.g. `@<commit SHA> # v2.0.0`.
        #       https://github.com/upsidr/merge-gatekeeper/tags
        #       https://github.com/upsidr/merge-gatekeeper/branches
        uses: upsidr/merge-gatekeeper@v2
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
```
<!-- == imptr: basic-yaml / end == -->

<!-- == export: simple-usage / end == -->

### Using Importer

You can also use the latest spec by using Importer to import directly from the sample setup in this repository.

Create a YAML file with just a single Importer Marker:

```yaml
# == imptr: merge-gatekeeper / begin from: https://github.com/upsidr/merge-gatekeeper/blob/main/example/definitions.yaml#[standard-setup] ==
# == imptr: merge-gatekeeper / end ==
```

With that, you can simply run `importer update FILENAME` to get the latest spec. You can also update the file used to specific branch or version.
