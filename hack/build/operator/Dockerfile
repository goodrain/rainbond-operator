FROM golang:1.13 as builder
ENV CGO_ENABLED=0
ENV GOPATH=/go

WORKDIR /go/src/github.com/goodrain/rainbond-operator
COPY pkg pkg
COPY cmd cmd
COPY version version
COPY vendor vendor

ARG GOARCH
ARG LDFLAGS
RUN go build -ldflags "$LDFLAGS" -o /opeator $PWD/cmd/manager

FROM alpine:3.11.2
RUN apk add --update tzdata \
    && mkdir /app \
    && apk add --update apache2-utils \
    && rm -rf /var/cache/apk/*
ENV TZ=Asia/Shanghai
COPY --from=builder opeator /app

ENTRYPOINT ["/app/opeator"]