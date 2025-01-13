TOKEN=${GITHUB_TOKEN}
REF=main
REPO=upsidr/gatekeeper
IGNORED=""

go-build:
	GO111MODULE=on LANG=en_US.UTF-8 CGO_ENABLED=0 go build ./cmd/gatekeeper

go-run: go-build
	./gatekeeper validate --token=$(TOKEN) --ref $(REF) --repo $(REPO) --ignored "$(IGNORED)"

docker-build:
	docker build -t gatekeeper:latest .

docker-run: docker-build
	docker run --rm -it --name gatekeeper gatekeeper:latest validate --token=$(TOKEN) --ref $(REF) --repo $(REPO) --ignored "$(IGNORED)"

test:
	go test ./...