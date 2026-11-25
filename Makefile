TOKEN=${GITHUB_TOKEN}
REF=main
REPO=upsidr/merge-gatekeeper
IGNORED=""

# Run the action locally against a real repository and ref.
run:
	GITHUB_REPOSITORY=$(REPO) INPUT_TOKEN=$(TOKEN) INPUT_REF=$(REF) INPUT_IGNORED="$(IGNORED)" node index.js

test:
	node --test
