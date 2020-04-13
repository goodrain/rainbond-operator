FROM golang:1.13 as builder
ENV CGO_ENABLED=0
ENV GOPATH=/go

WORKDIR /go/src/github.com/goodrain/rainbond-operator
COPY pkg pkg
COPY cmd cmd
COPY vendor vendor

ARG GOARCH
ARG LDFLAGS="-extldflags -static"
RUN go build -ldflags "$LDFLAGS" -o /openapi $PWD/cmd/openapi

FROM registry.cn-hangzhou.aliyuncs.com/goodrain/rbd-op-ui-base:v0.0.1-beta2
COPY --from=builder openapi /app
EXPOSE 8080

ENTRYPOINT ["/app/openapi"]