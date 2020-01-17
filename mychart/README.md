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

kubectl create ns rbd-system && helm install my-release ./mychart --namespace=rbd-system
```