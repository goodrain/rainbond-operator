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
COPY version version

ARG GOARCH
ARG LDFLAGS
RUN go build -ldflags "$LDFLAGS" -o /opeator $PWD/cmd/manager

FROM abewang/rbd-op-ui-base
RUN apk add --update tzdata
ENV TZ=Asia/Shanghai
COPY --from=builder opeator /app

ENTRYPOINT ["/app/opeator"]