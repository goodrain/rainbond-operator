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

FROM registry.cn-hangzhou.aliyuncs.com/goodrain/rbd-op-ui-base:d0f3862-dirty
COPY --from=builder openapi /app
ADD hack/version /app/version
EXPOSE 8080

ENTRYPOINT ["/app/openapi"]
