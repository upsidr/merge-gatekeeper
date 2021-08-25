# Merge Gatekeeper

Merge Gatekeeper provides extra control for Pull Request management.

## 🌄 What does Merge Gatekeeper provide, and Why?

<!-- == imptr: background / begin from: ./docs/details.md#[background] == -->

Pull Request plays a significant part in day-to-day development, and it is essential to ensure all merges are well controlled and managed to build robust system. GitHub provides some control over CI, reviews, etc., but there are some limitations with handling special cases.

At UPSIDER, we have a few internal repositories set up with a monorepo structure, where there are many types of code in a single repository. This comes with its own pros and cons, but one difficulty is how we end up with various CI jobs, which are only run for some changes that touch relevant files. With GitHub's branch protection,there is no way to specify "Ensure Go build and test pass _if and only if_ Go code is updated", or "Ensure E2E tests are run and successful _if and only if_ frontend code is updated". Because of this limitation, we would either need to run all the CI jobs for all the code for any Pull Requests, or do not set any limitation based on the CI status. <sup><sub><sup>(\*1)</sup></sub></sup>

**Merge Gatekeeper** was created to provide more control over merges. By placing Merge Gatekeeper for all PRs, it will check all other CI jobs that get kicked off for any PR, and ensures all the jobs are completed successfully. If there is any job that has failed, Merge Gatekeeper will fail as well. This allows you to control merge protection by looking at Merge Gatekeeper status, which means you can effectively ensure any CI that fails will block the PR merge. All you need is the Merge Gatekeeper as one of the PR based GitHub Action, and set the branch protection rule as shown below.

![Branch protection example](/assets/images/branch-protection-example.png)

---

<sup><sub>NOTE <sup>(\*1)</sup>: There are some other hacks, such as using an empty job with the same name to override the status, but those solutions do not provide the flexible control we are after.</sub></sup>

<!-- == imptr: background / end == -->

---

You can find [more details here](/docs/details.md).

## 🧪 Action Inputs

<!-- == imptr: inputs / begin from: ./docs/action-usage.md#[inputs] == -->

| Name       | Description                                                                                                                                                                                                                                                                                          | Required |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------: |
| `token`    | `GITHUB_TOKEN` or Personal Access Token with `repo` scope                                                                                                                                                                                                                                            |   Yes    |
| `self`     | The name of Merge Gatekeeper job, and defaults to `merge-gatekeeper`. This is used to check other job status, and do not check Merge Gatekeeper itself. If you updated the GitHub Action job name from `merge-gatekeeper` to something else, you would need to specify the new name with this value. |          |
| `interval` | Check interval to recheck the job status. Default is set to 30 (sec).                                                                                                                                                                                                                                |          |
| `timeout`  | Timeout setup to give up further check. Default is set to 600 (sec).                                                                                                                                                                                                                                 |          |
| `ref`      | Git ref to check out. This falls back to the HEAD for given PR, but can be set to any ref.                                                                                                                                                                                                           |          |

<!-- == imptr: inputs / end == -->

## 🚀 Example Usage

<!-- == imptr: example-usage / begin from: ./docs/action-usage.md#[simple-usage] == -->

The easiest approach is to copy the below definition, and save it under `.github/workspaces` directory. There is no further modification required unless you have some specific requirements.

<!-- TODO: replace below using Importer once Importer supports code block wrapping
     == imptr: basic-yaml / begin from: ./example/definitions.yaml#[standard-setup] == -->

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
    steps:
      - name: Run Merge Gatekeeper
        uses: upsidr/merge-gatekeeper@main
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
```

You can find the exact file at [`/example/merge-gatekeeper.yml`](/example/merge-gatekeeper.yml).

<!-- == imptr: example-usage / end == -->
