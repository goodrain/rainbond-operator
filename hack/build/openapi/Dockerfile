FROM golang:1.13 as builder
ENV CGO_ENABLED=0
ENV GOPATH=/go

WORKDIR /go/src/github.com/goodrain/rainbond-operator
COPY pkg pkg
COPY cmd cmd
COPY vendor vendor

ARG GOARCH
ARG LDFLAGS
RUN go build -ldflags "$LDFLAGS" -o /openapi $PWD/cmd/openapi

FROM IMAGE_DOMAIN/IMAGE_NAMESPACE/rbd-op-ui-base:VERSION
COPY --from=builder openapi /app
ADD hack/version /app/version
ENV RELEASE_DESC __RELEASE_DESC__
EXPOSE 8080

ENTRYPOINT ["/app/openapi"]
