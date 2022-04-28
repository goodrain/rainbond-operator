#!/bin/bash

IMAGE_DOMAIN=${IMAGE_DOMAIN:-'docker.io'}
IMAGE_NAMESPACE=${IMAGE_NAMESPACE:-'rainbond'}
DOMESTIC_BASE_NAME=${DOMESTIC_BASE_NAME:-'registry.cn-hangzhou.aliyuncs.com'}
DOMESTIC_NAMESPACE=${DOMESTIC_NAMESPACE:-'goodrain'}

imageName=${IMAGE_DOMAIN}/${IMAGE_NAMESPACE}/rainbond-operator:${VERSION}
docker build -t "${imageName}" -f Dockerfile .
echo "$DOCKER_PASSWORD" | docker login ${IMAGE_DOMAIN} -u "$DOCKER_USERNAME" --password-stdin
docker push ${imageName}

domestcName=${DOMESTIC_BASE_NAME}/${DOMESTIC_NAMESPACE}/rainbond-operator:${VERSION}
docker tag "${imageName}" "${domestcName}"
echo "${DOMESTIC_DOCKER_PASSWORD}"|docker login -u "${DOMESTIC_DOCKER_USERNAME}" "${DOMESTIC_BASE_NAME}" --password-stdin
docker push "${domestcName}"