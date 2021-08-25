ARG GO_VERSION=1.16.7

FROM golang:${GO_VERSION}-alpine

ENV GO111MODULE on
ENV LANG en_US.UTF-8
ENV ORG upsidr
ENV REPO merge-gatekeeper
ENV APP_NAME merge-gatekeeper

RUN mkdir -p $GOPATH/src

WORKDIR ${GOPATH}/src/github.com/${ORG}/${REPO}

COPY vendor .
COPY go.mod .
COPY go.sum .

WORKDIR ${GOPATH}/src/github.com/${ORG}/${REPO}/cmd
COPY cmd .

WORKDIR ${GOPATH}/src/github.com/${ORG}/${REPO}/internal
COPY internal .

WORKDIR ${GOPATH}/src/github.com/${ORG}/${REPO}

RUN CGO_ENABLED=0 go build -mod vendor ./cmd/${APP_NAME} \
    && mv ${APP_NAME} /go/bin/

ENTRYPOINT ["/go/bin/merge-gatekeeper"]
