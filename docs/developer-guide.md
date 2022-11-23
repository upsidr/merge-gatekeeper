# Working with Merge Gatekeeper locally

## Requirements

- [`go` >= 1.16.7](https://go.dev/doc/install)
- [A valid github token with `repo` permissions](https://github.com/settings/tokens)
- [make](https://en.wikipedia.org/wiki/Make_(software))

## Useful but not required

- [`docker`](https://docs.docker.com/engine/install/)
  - required for building and running via docker

## Building Merge Gatekeeper

Using the [`Makefile`](./../Makefile) run the following to build:
```bash
# build go binary
make go-build

# build docker container
make docker-build
```

## Running Merge Gatekeeper

it is recommend to export you [github token with `repo` permissions](https://github.com/settings/tokens) to the environment using
```bash
export GITHUB_TOKEN="your token"
```
otherwise, you will need to pass your token in via
```bash
GITHUB_TOKEN="your token" make go-run
```

Using the [`Makefile`](./../Makefile) run the following to run:
```bash
# build and run go binary
make go-run

# build and run docker container
make docker-run
```

## Testing
To test, use the makefile:

```bash
make test
```
