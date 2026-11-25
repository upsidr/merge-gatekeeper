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

- Runtime code (`index.js`, `src/`) uses **Node.js built-ins only**, on the `node24` runtime that the Actions runner bundles. There is no `node_modules`, no bundler and no `dist/`: the files in this repository are exactly what runs. ESLint enforces this (`no-restricted-imports`).
- Do not add a runtime dependency. If something seems to need one, look for a built-in first and discuss it in an issue.
- Tools used only in CI (ESLint, CodeQL, importer) are fetched in CI, pinned to an **exact version that has been public for at least two weeks**, and go through the Takumi Guard proxy (`npm_config_registry=https://npm.flatt.tech`, `GOPROXY=https://golang.flatt.tech`).
- Pin every third-party action in `.github/workflows` to a **full commit SHA** with the version in a trailing comment, and give each job the minimum `permissions` it needs.

## Security expectations

- Never log the token, or a proxy URL (it can contain credentials). Log variable names instead.
- Treat everything from the GitHub API (job names, messages) as untrusted. Escape it before putting it into a workflow command (`::error::`, `::warning::`); see `escapeData` in `index.js`.
- No `eval`, `new Function`, or `child_process`. The only network traffic is to the GitHub REST API (`GITHUB_API_URL`), through the proxy settings of the environment.
- SAST runs on every pull request: CodeQL (`.github/workflows/codeql.yaml`, `actions` and `javascript-typescript`) and Wiz scanners (configured outside this repository). Fix findings rather than dismissing them; if a finding is a false positive, explain why in the pull request.

## Before opening a pull request

```bash
node --test --test-timeout=30000   # or: make test
npm run lint                       # or: make lint (ESLint via npx, pinned)
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
- For a stack of pull requests, merge from the bottom. After each merge, rebase the next one onto `main` and change its base to `main`; CI runs on that push.

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
