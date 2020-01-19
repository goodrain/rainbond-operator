# Goodrain rainbond-operator

[rainbond-operator](https://github.com/GLYASAI/rainbond-operator) Simplify rainbond cluster
configuration and management.

__DISCLAIMER:__ While this chart has been well-tested, the rainbond-operator is still currently in beta.
Current project status is available [here](https://github.com/GLYASAI/rainbond-operator).

## Introduction

This chart bootstraps an rainbond-operator.

## Prerequisites

- Kubernetes 1.2+
- Helm 3.0+

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
git clone https://github.com/GLYASAI/rainbond-operator.git

cd rainbond-operator

kubectl create ns rbd-system

helm install my-release ./mychart --namespace=rbd-system
```

## Uninstalling the Chart

To uninstall/delete the my-release:

```bash
helm delete my-release
```

The command removes all the Kubernetes components EXCEPT the persistent volume.

## Configuration

The following table lists the configurable parameters of the etcd-operator chart and their default values.

| Parameter                           | Description                                   | Default                                                                                                                                                                  |
|-------------------------------------|-----------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `rainbondOperator.name`             | Rainbond Operator name                        | `rainbond-operator`                                                                                                                                                      |
| `rainbondOperator.image.repository` | rainbond-operator container image             | `abewang/rainbond-operator`                                                                                                                                              |
| `rainbondOperator.image.tag`        | rainbond-operator container image tag         | `v0.0.1`                                                                                                                                                                 |
| `rainbondOperator.image.pullPolicy` | rainbond-operator container image pull policy | `IfNotPresent`                                                                                                                                                           |
| `openapi.name`                      | openapi name                                  | `openapi`                                                                                                                                                                |
| `openapi.image.repository`          | openapi container image                       | `abewang/rbd-op-ui`                                                                                                                                                      |
| `openapi.image.tag`                 | openapi container image tag                   | `v0.0.1`                                                                                                                                                                 |
| `openapi.image.pullPolicy`          | openapi container image pull policy           | `IfNotPresent`                                                                                                                                                           |
| `openapi.image.port`                | openapi service port                          | `8080`                                                                                                                                                                   |
| `openapi.image.nodePort`            | openapi service nodePort                      | `30008`                                                                                                                                                                  |
| `openapi.image.downloadURL`         | rainbond package download url                 | `https://hrhtest.oss-cn-shanghai.aliyuncs.com/rainbond-pkg-V5.2-dev.tgz?OSSAccessKeyId=LTAIVsBmV7qjFJzK&Expires=1579407682&Signature=2Nmf5ZBAGIo%2F05%2BogDyAgSaSJNI%3D` |


Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```bash
$ helm install --name my-release --set image.tag=v0.0.1 ./mychart
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while
installing the chart. For example:

```bash
$ helm install --name my-release --values values.yaml ./mychart
```