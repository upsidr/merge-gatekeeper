# Action Details

## Action Inputs

<!-- == export: inputs / begin == -->

| Name       | Description                                                                                                                                                                                                                                                                              | Required |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------: |
| `token`    | `GITHUB_TOKEN` or Personal Access Token with `repo` scope                                                                                                                                                                                                                                |   Yes    |
| `self`     | The name of Merge Gatekeeper job, and defaults to `gatekeeper`. This is used to check other job status, and do not check Merge Gatekeeper itself. If you updated the GitHub Action job name from `gatekeeper` to something else, you would need to specify the new name with this value. |          |
| `interval` | Check interval to recheck the job status. Default is set to 5 (sec).                                                                                                                                                                                                                     |          |
| `timeout`  | Timeout setup to give up further check. Default is set to 600 (sec).                                                                                                                                                                                                                     |          |
| `ignored`  | Jobs to ignore regardless of their statuses. Defined as a comma-separated list.                                                                                                                                                                                                          |          |
| `ref`      | Git ref to check out. This falls back to the HEAD for given PR, but can be set to any ref.                                                                                                                                                                                               |          |

<!-- == export: inputs / end == -->

## Usage

### Copy Standard YAML

<!-- == export: simple-usage / begin == -->

The easiest approach is to copy the standard definition, and save it under `.github/workflows` directory. There is no further modification required unless you have some specific requirements.

#### With `curl`

```bash
curl -sSL https://raw.githubusercontent.com/upsidr/gatekeeper/main/example/gatekeeper.yml \
  > .github/workflows/gatekeeper.yml
```

#### Directly copy YAML

The below is the copy of [`/example/gatekeeper.yml`](/example/gatekeeper.yml), with extra comments.

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
  gatekeeper:
    runs-on: ubuntu-latest
    # Restrict permissions of the GITHUB_TOKEN.
    # Docs: https://docs.github.com/en/actions/using-jobs/assigning-permissions-to-jobs
    permissions:
      checks: read
      statuses: read
    steps:
      - name: Run Merge Gatekeeper
        # NOTE: v1 is updated to reflect the latest v1.x.y. Please use any tag/branch that suits your needs:
        #       https://github.com/argandtech/gatekeeper/tags
        #       https://github.com/argandtech/gatekeeper/branches
        uses: upsidr/gatekeeper@v1
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
```
<!-- == imptr: basic-yaml / end == -->

<!-- == export: simple-usage / end == -->

### Using Importer

You can also use the latest spec by using Importer to improt directly from the sample setup in this repository.

Create a YAML file with just a single Importer Marker:

```yaml
# == imptr: gatekeeper / begin from: https://github.com/argandtech/gatekeeper/blob/main/example/definitions.yaml#[standard-setup] ==
# == imptr: gatekeeper / end ==
```

With that, you can simply run `importer update FILENAME` to get the latest spec. You can also update the file used to specific branch or version.

###
