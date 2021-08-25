# Merge Gatekeeper

Merge Gatekeeper provides extra control for Pull Request management.

## 🌄 What does Merge Gatekeeper provide, and Why?

<!-- == imptr: background / begin from: ./docs/details.md#[background] == -->

Pull Request plays a significant part in day-to-day development, and making sure all the merges are controled is essential for building robust system. GitHub provides some control over CI, reviews, etc., but there are some limitations with handling special cases.

At UPSIDER, we have a few internal repositories set up with a monorepo structure, where there are many types of code in the single repository. This comes with its own pros and cons, but with GitHub Action and merge control, there is no way to specify "Ensure Go build and test pass _if and only if_ Go code is updated", or "Ensure E2E tests are run and successful _if and only if_ frontend code is updated". Because of this limitation, we would either need to run all the CI jobs for all the code for any Pull Requests, or do not set any limitation based on the CI status. <sup>(\*1)</sup>

Merge Gatekeeper was created to provide more control over merges.

---

NOTE <sup>(\*1)</sup>: There are some other hacks, such as using an empty job with the same name to override the status, but those solutions do not provide the flexible control we are after.

<!-- == imptr: background / end == -->

---

You can find [more details here](/details.md).

## 🧪 Action Inputs

<!-- == imptr: inputs / begin from: ./docs/action-usage.md#[inputs] == -->

| Name       | Description                                                                                | Required |
| ---------- | ------------------------------------------------------------------------------------------ | :------: |
| `token`    | `GITHUB_TOKEN` or Personal Access Token with `repo` scope                                  |   Yes    |
| `job`      | Target job to check against. If none is provided, it will fall back to check all the jobs. |          |
| `interval` | Check interval for merge-gatekeeper to recheck the job status. Default is set to 30 (sec). |          |
| `timeout`  | Timeout setup for merge-gatekeeper to give up further check. Default is set to 600 (sec).  |          |
| `ref`      | Git ref to check out. This falls back to the HEAD for given PR, but can be set to any ref. |          |

<!-- == imptr: inputs / end == -->
