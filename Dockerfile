FROM golang:alpine AS builder
LABEL maintainer="Mikolaj Gasior"

RUN apk add --update git bash openssh make

WORKDIR /go/src/github.com/mikogs/grafana-sidecar-users-yaml
COPY . .
RUN go build

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /bin
COPY --from=builder /go/src/github.com/mikogs/grafana-sidecar-users-yaml/grafana-sidecar-users-yaml .

ENTRYPOINT ["/bin/grafana-sidecar-users-yaml"]
