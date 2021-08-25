ARG GO_VERSION=1.16.7

FROM golang:${GO_VERSION}-alpine

ENV GO111MODULE on
ENV LANG en_US.UTF-8
ENV ORG upsidr
ENV REPO merge-gatekeeper
ENV APP_NAME merge-gatekeeper

RUN mkdir -p $GOPATH/src

WORKDIR ${GOPATH}/src/github.com/${ORG}/${REPO}

COPY vendor go.mod go.sum cmd internal .

RUN CGO_ENABLED=0 go build ./cmd/${APP_NAME} \
    && mv ${APP_NAME} /go/bin/

ENTRYPOINT ["/go/bin/merge-gatekeeper"]
