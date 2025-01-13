ARG GO_VERSION=1.23.0

FROM golang:${GO_VERSION}-alpine

ARG ORG=upsidr
ARG REPO=gatekeeper

ENV GO111MODULE=on LANG=en_US.UTF-8

RUN mkdir -p $GOPATH/src

WORKDIR ${GOPATH}/src/github.com/${ORG}/${REPO}

COPY . .

RUN CGO_ENABLED=0 go build . \
  && mv gatekeeper /go/bin/

ENTRYPOINT ["/go/bin/gatekeeper"]
