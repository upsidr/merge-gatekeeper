# Action Details

## Action Inputs

<!-- == export: inputs / begin == -->

| Name       | Description                                                                                                                                                                                                                                                                                          | Required |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------: |
| `token`    | `GITHUB_TOKEN` or Personal Access Token with `repo` scope                                                                                                                                                                                                                                            |   Yes    |
| `self`     | The name of Merge Gatekeeper job, and defaults to `merge-gatekeeper`. This is used to check other job status, and do not check Merge Gatekeeper itself. If you updated the GitHub Action job name from `merge-gatekeeper` to something else, you would need to specify the new name with this value. |          |
| `interval` | Check interval to recheck the job status. Default is set to 5 (sec).                                                                                                                                                                                                                                 |          |
| `timeout`  | Timeout setup to give up further check. Default is set to 600 (sec).                                                                                                                                                                                                                                 |          |
| `initial-delay` | Delay in seconds before the first validation. Useful when other jobs are known to take a while, to avoid needless polling at the start. This delay is not counted towards `timeout`. Default is set to 0 (sec).                                                                                      |          |
| `ignored`  | Jobs to ignore regardless of their statuses. Defined as a comma-separated list.                                                                                                                                                                                                                      |          |
| `ref`      | Git ref to check out. This falls back to the HEAD for given PR, but can be set to any ref.                                                                                                                                                                                                           |          |

<!-- == export: inputs / end == -->

## Runner Requirements

<!-- == export: runner / begin == -->

Merge Gatekeeper is a JavaScript action that runs on the `node24` runtime bundled with the GitHub Actions runner. It does not need Docker, and it does not need `actions/checkout` or `actions/setup-node`.

- **We recommend `runs-on: ubuntu-latest`.** The job mostly waits for other jobs, so it needs no particular machine. Linux runners start fastest and are the cheapest GitHub-hosted runners, while macOS and Windows runners are billed at a higher rate for the same wait. Merge Gatekeeper also works on them if you have a reason to use them.
- **Self-hosted runners** must run a version of the Actions runner that supports `node24` actions. Docker is not required, so runners inside containers or without Docker access work as well.
- **GitHub Enterprise Server** is supported. The API endpoint is taken from `GITHUB_API_URL`, which the runner sets automatically.
- **Network access** to the GitHub REST API (`api.github.com`, or your Enterprise Server API) is the only outbound connection Merge Gatekeeper makes.

<!-- == export: runner / end == -->

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
    runs-on: ubuntu-latest
    # Restrict permissions of the GITHUB_TOKEN.
    # Docs: https://docs.github.com/en/actions/using-jobs/assigning-permissions-to-jobs
    permissions:
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
