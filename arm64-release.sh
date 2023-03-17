#! /bin/bash

export NAMESPACE=wutong-operator
export VERSION=v1.1.0-stable-arm64

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -o ./bin/app main.go
docker build . -t swr.cn-southwest-2.myhuaweicloud.com/wutong/${NAMESPACE}:${VERSION}
docker push swr.cn-southwest-2.myhuaweicloud.com/wutong/${NAMESPACE}:${VERSION}

# docker tag swr.cn-southwest-2.myhuaweicloud.com/wutong/${NAMESPACE}:${VERSION} wutongpaas/${NAMESPACE}:${VERSION}
# docker push wutongpaas/${NAMESPACE}:${VERSION}