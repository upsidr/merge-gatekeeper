# Action Details

## Action Inputs

<!-- == export: inputs / begin == -->

| Name                  | Description                                                                                                                                                                                                                                                                                          | Required |
| --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------: |
| `token`               | `GITHUB_TOKEN` or Personal Access Token with `repo` scope                                                                                                                                                                                                                                            |   Yes    |
| `self`                | The name of Merge Gatekeeper job, and defaults to `merge-gatekeeper`. This is used to check other job status, and do not check Merge Gatekeeper itself. If you updated the GitHub Action job name from `merge-gatekeeper` to something else, you would need to specify the new name with this value. |          |
| `interval`            | Check interval to recheck the job status. Default is set to 5 (sec).                                                                                                                                                                                                                                 |          |
| `timeout`             | Timeout setup to give up further check. Default is set to 600 (sec).                                                                                                                                                                                                                                 |          |
| `ignored`             | Jobs to ignore regardless of their statuses. Defined as a comma-separated list.                                                                                                                                                                                                                      |          |
| `ref`                 | Git ref to check out. This falls back to the HEAD for given PR, but can be set to any ref.                                                                                                                                                                                                           |          |
| `github-client-retry` | Retry the request if the GitHub client returns 5xx error. Default is set to 0.                                                                                                                                                                                                                       |          |

<!-- == export: inputs / end == -->

## Usage

### Copy Standard YAML

<!-- == export: simple-usage / begin == -->

The easiest approach is to copy the standard definition, and save it under `.github/workspaces` directory. There is no further modification required unless you have some specific requirements.

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
        # NOTE: v1 is updated to reflect the latest v1.x.y. Please use any tag/branch that suits your needs:
        #       https://github.com/upsidr/merge-gatekeeper/tags
        #       https://github.com/upsidr/merge-gatekeeper/branches
        uses: upsidr/merge-gatekeeper@v1
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
```
<!-- == imptr: basic-yaml / end == -->

<!-- == export: simple-usage / end == -->

### Using Importer

You can also use the latest spec by using Importer to improt directly from the sample setup in this repository.

Create a YAML file with just a single Importer Marker:

```yaml
# == imptr: merge-gatekeeper / begin from: https://github.com/upsidr/merge-gatekeeper/blob/main/example/definitions.yaml#[standard-setup] ==
# == imptr: merge-gatekeeper / end ==
```

With that, you can simply run `importer update FILENAME` to get the latest spec. You can also update the file used to specific branch or version.

###
