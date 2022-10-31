ARG GO_VERSION=1.16.7

FROM golang:${GO_VERSION}-alpine

ARG ORG=dispatchhealth
ARG REPO=merge-gatekeeper

RUN mkdir -p $GOPATH/src

WORKDIR ${GOPATH}/src/github.com/${ORG}/${REPO}

COPY . .

RUN make build \
    && mv merge-gatekeeper /go/bin/

ENTRYPOINT ["/go/bin/merge-gatekeeper"]
