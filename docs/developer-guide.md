# Working with Merge Gatekeeper locally

Merge Gatekeeper is a dependency-free JavaScript action. There is nothing to install or build: `index.js` runs directly on Node.js.

## Requirements

- [Node.js >= 24](https://nodejs.org/)
- [A valid github token with `repo` permissions](https://github.com/settings/tokens)
- [make](https://en.wikipedia.org/wiki/Make_(software)) (optional)

## Layout

- `index.js`: action entry point, referenced from `action.yml`
- `src/core.js`: pure decision logic (API responses → per-job states → verdict, and the log summary)
- `src/github.js`: GitHub REST client using the built-in `fetch`
- `src/run.js`: reads inputs and polls until success, failure, or timeout
- `test/`: tests using `node:test`

## Running Merge Gatekeeper

It is recommended to export your [github token with `repo` permissions](https://github.com/settings/tokens) to the environment using

```bash
export GITHUB_TOKEN="your token"
```

otherwise, you will need to pass your token in via

```bash
GITHUB_TOKEN="your token" make run
```

Using the [`Makefile`](./../Makefile), run the following to check a ref of a repository:

```bash
make run REPO=upsidr/merge-gatekeeper REF=main
```

## Testing

```bash
make test
# or
node --test
```
