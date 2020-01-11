FROM golang:1.13 as builder
ENV GO111MODULE=on
ENV CGO_ENABLED=0
ENV GOPATH=/go
ENV GOPROXY=https://goproxy.cn

WORKDIR /go/src/github.com/GLYASAI/rainbond-operator
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY pkg pkg
COPY cmd cmd

ARG GOARCH
ARG LDFLAGS
RUN go build -ldflags "$LDFLAGS" -o /openapi $PWD/cmd/openapi

FROM alpine:3.11.2
RUN apk add --update tzdata
ENV TZ=Asia/Shanghai
COPY --from=builder openapi /
EXPOSE 8080

ENTRYPOINT ["/openapi"]