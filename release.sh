#!/bin/bash

IMAGE_DOMAIN=${IMAGE_DOMAIN:-'docker.io'}
IMAGE_NAMESPACE=${IMAGE_NAMESPACE:-'rainbond'}
DOMESTIC_BASE_NAME=${DOMESTIC_BASE_NAME:-'registry.cn-hangzhou.aliyuncs.com'}
DOMESTIC_NAMESPACE=${DOMESTIC_NAMESPACE:-'goodrain'}
ARCH=${BUILD_ARCH:-'amd64'}

imageName=${IMAGE_DOMAIN}/${IMAGE_NAMESPACE}/rainbond-operator:${VERSION}
docker build --build-arg ARCH="${ARCH}" -t "${imageName}" -f Dockerfile .

if [ "$DOCKER_USERNAME" ]; then
  echo "$DOCKER_PASSWORD" | docker login ${IMAGE_DOMAIN} -u "$DOCKER_USERNAME" --password-stdin
  docker push "${imageName}"
fi

if [ "$DOMESTIC_DOCKER_USERNAME" ]; then
  domestcName=${DOMESTIC_BASE_NAME}/${DOMESTIC_NAMESPACE}/rainbond-operator:${VERSION}
  docker tag "${imageName}" "${domestcName}"
  echo "${DOMESTIC_DOCKER_PASSWORD}"|docker login -u "${DOMESTIC_DOCKER_USERNAME}" "${DOMESTIC_BASE_NAME}" --password-stdin
  docker push "${domestcName}"
fi