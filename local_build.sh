export NAMESPACE=wutong-operator
export VERSION=v1.1.0-stable

CGO_ENABLED=0 GO111MODULE=on go build -a -o ./bin/app main.go
docker build . -t swr.cn-southwest-2.myhuaweicloud.com/wutong/${NAMESPACE}:${VERSION} 
docker push swr.cn-southwest-2.myhuaweicloud.com/wutong/${NAMESPACE}:${VERSION}