# Build the manager binary
FROM --platform=$BUILDPLATFORM golang:1.15 as builder
ARG TARGETOS TARGETPLATFORM
WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
ENV GOPROXY=https://goproxy.io
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY util util/

# Build
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETPLATFORM GO111MODULE=on go build -a -o manager main.go


FROM wutongpaas/alpine:3.15
RUN mkdir /app \
    && apk add --update apache2-utils \
    && rm -rf /var/cache/apk/*
ENV TZ=Asia/Shanghai
WORKDIR /
COPY --from=builder /workspace/manager .

CMD ["/manager"]
