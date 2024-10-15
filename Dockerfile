# Build the manager binary
FROM golang:1.18 as builder
ARG TARGETARCH
ARG GOPROXY
WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
ENV GOPROXY=$GOPROXY
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY util util/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH="${TARGETARCH}" GO111MODULE=on go build -a -o manager main.go


FROM alpine:3.11.2
RUN apk add --update tzdata \
    && mkdir /app \
    && apk add --update apache2-utils \
    && rm -rf /var/cache/apk/*
ENV TZ=Asia/Shanghai
WORKDIR /

COPY config/prom config/prom/

COPY --from=builder /workspace/manager .
CMD ["/manager"]
