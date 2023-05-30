# Merge Gatekeeper

Merge Gatekeeper provides extra control for Pull Request management.

## 🌄 What Does Merge Gatekeeper Provide, and Why?

<!-- == imptr: background / begin from: ./docs/details.md#[background] == -->

Pull Request plays a significant part in day-to-day development, and it is essential to ensure all merges are well controlled and managed to build robust system. GitHub provides controls over CI, reviews, etc., but there are some limitations around handling specific use cases. Merge Gatekeeper helps overcome those by adding extra controls, such as monorepo friendly branch protection.

At UPSIDER, we have a few internal repositories set up with a monorepo structure, with many types of code in a single repository. This comes with its own pros and cons, but one difficulty is how we end up with various CI jobs, which only run for changes that touch relevant files. With GitHub's branch protection, there is no way to specify "Ensure Go build and test pass _if and only if_ Go code is updated", or "Ensure E2E tests are run and successful _if and only if_ frontend code is updated". This is due to the GitHub branch protection design to specify a list of jobs to pass, which is only driven by the target branch name, regardless of change details. Because of this limitation, we would either need to run all the CI jobs for any Pull Requests, or do not set any limitation based on the CI status. <sup><sub><sup>(\*1)</sup></sub></sup>

**Merge Gatekeeper** was created to provide more control for merges. By placing Merge Gatekeeper to run for all PRs, it can check all other CI jobs that get kicked off, and ensure all the jobs are completed successfully. If there is any job that has failed, Merge Gatekeeper will fail as well. This allows merge protection based on Merge Gatekeeper, which can effectively ensure any CI failure will block merge. All you need is the Merge Gatekeeper as one of the PR based GitHub Action, and set the branch protection rule as shown below.

![Branch protection example](/assets/images/branch-protection-example.png)

We are looking to add a few more features, such as extra signoff from non-coder, label based check, etc.

<sup><sub>NOTE:
<sup>(\*1)</sup> There are some other hacks, such as using an empty job with the same name to override the status, but those solutions do not provide the flexible control we are after.</sub></sup>

<!-- == imptr: background / end == -->

You can find [more details here](/docs/details.md).

## 🚀 How Can I Use Merge Gatekeeper?

<!-- == imptr: example-usage / begin from: ./docs/action-usage.md#[simple-usage] == -->

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

<!-- == imptr: example-usage / end == -->

You can find [more details here](/docs/action-usage.md).

## 🧪 Action Inputs

There are some customisation available for Merge Gatekeeper.

<!-- == imptr: inputs / begin from: ./docs/action-usage.md#[inputs] == -->

| Name       | Description                                                                                                                                                                                                                                                                                          | Required |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------: |
| `token`    | `GITHUB_TOKEN` or Personal Access Token with `repo` scope                                                                                                                                                                                                                                            |   Yes    |
| `self`     | The name of Merge Gatekeeper job, and defaults to `merge-gatekeeper`. This is used to check other job status, and do not check Merge Gatekeeper itself. If you updated the GitHub Action job name from `merge-gatekeeper` to something else, you would need to specify the new name with this value. |          |
| `interval` | Check interval to recheck the job status. Default is set to 5 (sec).                                                                                                                                                                                                                                 |          |
| `github-client-retry` | Retry the request if the GitHub client returns 5xx error. Default is set to 0.                                                                                                                                                                                                                                 |          |
| `timeout`  | Timeout setup to give up further check. Default is set to 600 (sec).                                                                                                                                                                                                                                 |          |
| `ignored`  | Jobs to ignore regardless of their statuses. Defined as a comma-separated list.                                                                                                                                                                                                                      |          |
| `ref`      | Git ref to check out. This falls back to the HEAD for given PR, but can be set to any ref.                                                                                                                                                                                                           |          |

<!-- == imptr: inputs / end == -->

You can find [more details here](/docs/action-usage.md).
