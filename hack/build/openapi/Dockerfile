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

FROM registry.cn-hangzhou.aliyuncs.com/goodrain/rbd-op-ui-base:v1.0.0
COPY --from=builder openapi /app
ENV RELEASE_DESC __RELEASE_DESC__
EXPOSE 8080

ENTRYPOINT ["/app/openapi"]
