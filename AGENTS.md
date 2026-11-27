# AGENTS.md

Guidance for AI agents (and humans) maintaining Merge Gatekeeper.

Merge Gatekeeper is a **public, open source GitHub Action** (MIT). It is used by UPSIDER and by people outside UPSIDER, who pin it in their workflows. UPSIDER maintains it for as long as it uses it. Every change ships to users we cannot see, so compatibility and clarity matter more than speed.

## Everything here is public

- Issues, pull requests, commits, code comments, test fixtures and release notes are visible to anyone. Write them in **English**.
- Do not mention internal information: internal repository names, Slack links or messages, ticket IDs, internal URLs, people's names, customer data, or numbers taken from internal systems.
- When an internal repository is the reason for a change, describe the situation in general terms ("several workflows define a job with the same name") and build test data from scratch.
- Do not mention GitHub users with `@` unless you mean to notify them.

## Compatibility is the contract

- The **inputs and their defaults** in `action.yml` are the public API. Do not rename or remove an input, change a default, or make a valid value invalid without a major release.
- New behaviour must be opt-in, or a fix of clearly wrong behaviour (such as a failure being missed). Call out any change in how jobs are judged in the release notes, even in a minor release.
- Accept old input formats when adding new ones (for example, a bare number still means seconds now that `10m` is accepted).
- The log summary (`N out of M`, the `::group::` lists) is read by people and sometimes by scripts. Keep its format unless there is a reason to change it.
- Older major versions keep working: never move or delete their tags.

## The action has no dependencies

- Runtime code (`index.js`, `src/`) uses **Node.js built-ins only**, on the `node24` runtime that the Actions runner bundles. There are no runtime dependencies, no bundler and no `dist/`: the files in this repository are exactly what runs. ESLint enforces this (`no-restricted-imports`).
- Do not add a runtime dependency (`dependencies` in `package.json`). If something seems to need one, look for a built-in first and discuss it in an issue.
- Dev tools (ESLint) are `devDependencies`, pinned with an exact version and locked, including their own dependencies, in `package-lock.json`. Install them with `npm ci`. `.npmrc` sends every fetch through the Takumi Guard proxy and disables install scripts.
- Use versions that have been public for **at least two weeks**. When adding or updating a dev tool, resolve the lockfile with `npm install --before=<date two weeks ago>` so its dependencies follow the same rule.
- Other tools used in CI (CodeQL, importer) are pinned the same way; Go tools go through `GOPROXY=https://golang.flatt.tech`.
- Every third-party action in `.github/workflows` is pinned to a commit in the [workflow lockfile](https://docs.github.com/en/actions/concepts/security/workflow-dependency-lockfiles) `.github/workflows/actions.lock`. Reference actions by a full version tag (`@v6.0.3`). After adding, removing or updating a `uses:`, run `gh actions-lock --ignore-local-actions` (extension `github/gh-actions-lock`) and commit both files. Never edit the lockfile by hand. (`--ignore-local-actions` is for the self-test step `uses: ./`, which runs the checked-out action; the action itself uses no other actions.)
- Give each workflow the minimum `permissions` it needs, and use `persist-credentials: false` on `actions/checkout`.

## Security expectations

- Never log the token, or a proxy URL (it can contain credentials). Log variable names instead.
- Treat everything from the GitHub API (job names, messages) as untrusted. Escape it before putting it into a workflow command (`::error::`, `::warning::`); see `escapeData` in `index.js`.
- No `eval`, `new Function`, or `child_process`. The only network traffic is to the GitHub REST API (`GITHUB_API_URL`), through the proxy settings of the environment.
- SAST runs on every pull request: CodeQL (`.github/workflows/codeql.yaml`, `actions` and `javascript-typescript`) and Wiz scanners (configured outside this repository). Fix findings rather than dismissing them; if a finding is a false positive, explain why in the pull request.

## Before opening a pull request

```bash
node --test --test-timeout=30000   # or: make test
npm ci && npm run lint             # or: make lint (ESLint, pinned in package-lock.json)
find . -name '*.md' -exec importer update {} \;   # docs sync, see below
```

- Add tests for every behaviour change. The decision logic in `src/core.js` is pure: test it with plain data in `test/core.test.js`. When changing how jobs are judged, include the case from the issue.
- Base claims on evidence. Before saying an issue is caused by X, reproduce it or look at the API data; say so when something is still a hypothesis.

## Documentation

- `README.md` pulls sections from `docs/` and `example/definitions.yaml` with [Importer](https://github.com/upsidr/importer) markers (`<!-- == imptr: ... == -->`). Edit the source section (`docs/*.md` `export` markers, `example/definitions.yaml`), then run `importer update`. CI fails if the README is out of sync.
- When inputs change, update `action.yml`, the inputs table in `docs/action-usage.md`, and the examples together.

## Issues and pull requests from others

- Reply in English, thank the reporter, and explain the outcome. When closing, say why and link the release or pull request that resolved it.
- If a request still makes sense but the pull request no longer applies (for example, it changes code that was removed), close it with an explanation and keep tracking the idea in its issue.
- Keep changes to someone else's pull request minimal, and do not push to their branch without asking.

## Releases

- Versions follow semver. `version.txt` holds the version being released.
- After the release pull request is merged (squash merge), tag the merge commit with `vX.Y.Z`, `vX.Y` and `vX`, all lightweight tags on the same commit. Push the tags right away: the examples point at `@vX`.
- Create a GitHub Release for `vX.Y.Z` with these sections: summary (and "Before Upgrading" for breaking changes), 🎁 Features, 🐞 Bug Fixes, 📝 Documentation, 🔬 Misc, and a Full Changelog link. Include every change since the previous release, not only the latest pull request.
- Releases are [immutable](https://docs.github.com/en/code-security/supply-chain-security/understanding-your-software-supply-chain/immutable-releases) (enabled from v2.1.0), and a tag ruleset blocks moving or deleting `v*.*.*` tags. Once a release is published, its `vX.Y.Z` tag can never point at another commit, and the version number cannot be reused even if the release is deleted. Check the tag's commit before publishing; create the release as a draft first if anything needs a second look. The title and notes stay editable. `vX.Y` and `vX` are not part of a release and move forward as usual.
- A [stack of pull requests](https://docs.github.com/en/pull-requests/reference/stacked-pull-requests) merges from the bottom. GitHub rebases the remaining pull requests onto `main` and retargets them automatically.

## Layout

| Path | What it is |
| --- | --- |
| `action.yml` | Action metadata: inputs and the `node24` entry point |
| `index.js` | Entry point: proxy setup, error reporting |
| `src/core.js` | Pure decision logic and the log summary |
| `src/github.js` | GitHub REST client (`fetch`, paging, retries) |
| `src/run.js` | Reads inputs, polls until success, failure or timeout |
| `src/proxy.js` | `HTTPS_PROXY` / `NO_PROXY` support for `fetch` |
| `test/` | `node:test` tests |
| `docs/`, `example/` | Documentation sources for the README |
