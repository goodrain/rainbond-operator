GROUP=rainbond
VERSION=v1alpha1
IMAGE_DOMAIN=registry.cn-hangzhou.aliyuncs.com
IMAGE_NAMESPACE=goodrain
TAG=v0.0.1

.PHONY: gen
gen: crds-gen openapi-gen sdk-gen
crds-gen:
	operator-sdk generate crds
openapi-gen:
	# Build the latest openapi-gen from source
	which ./bin/openapi-gen > /dev/null || go build -o ./bin/openapi-gen k8s.io/kube-openapi/cmd/openapi-gen
    # Run openapi-gen for each of your API group/version packages
	./bin/openapi-gen --logtostderr=true \
    -o "" -i ./pkg/apis/$(GROUP)/$(VERSION) \
    -O zz_generated.openapi \
    -p ./pkg/apis/$(GROUP)/$(VERSION) \
    -h ./hack/k8s/codegen/boilerplate.go.txt -r "-"
sdk-gen:
	chmod +x vendor/k8s.io/code-generator/generate-groups.sh
	./hack/k8s/codegen/update-generated.sh
sdk-verify:
	./hack/k8s/codegen/verify-generated.sh

api-add:
	operator-sdk add api --api-version=rainbond.io/$(VERSION) --kind=$(KIND)

ctrl-add:
	operator-sdk add controller --api-version=rainbond.io/$(VERSION) --kind=$(KIND)

.PHONY: check
check:
	which ./bin/golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.23.2
	@bin/golangci-lint run



.PHONY: mock
mock:
	./mockgen.sh

.PHONY: build
build-ui:
	docker build . -f hack/ui/Dockerfile -t $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui-base:$(TAG)
build-api:
	docker build . -f hack/openapi/Dockerfile -t $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui:$(TAG)
build-api-dev:
	docker build . -f hack/openapi/Dockerfile.dev -t $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui:$(TAG)
build-operator:
	docker build . -f hack/operator/Dockerfile -t $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rainbond-operator:$(TAG)
build-operator-dev:
	docker build . -f hack/operator/Dockerfile.dev -t $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rainbond-operator:$(TAG)	
build: build-ui build-api build-operator

docker-login:
	docker login $(IMAGE_DOMAIN) -u $(DOCKER_USER) -p $(DOCKER_PASS)

.PHONY: push
push-ui: build-ui
	docker push $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui-base:$(TAG)
push-api: build-api
	docker push $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui:$(TAG)
push-operator: build-operator
	docker push $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rainbond-operator:$(TAG)
push: docker-login push-ui push-api push-operator

.PHONY: test
test-operator:build-operator
	docker save -o /tmp/rainbond-operator.tgz  $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rainbond-operator:$(TAG)
	scp /tmp/rainbond-operator.tgz root@172.20.0.20:/root
test-api:
	GOOS=linux go build -o openapi ./cmd/openapi
	docker build --no-cache . -f hack/openapi/Dockerfile.dev -t  $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui:$(TAG)
	rm -rf ./openapi

chart:
	tar -cvf rainbond-operator-chart.tar ./mychart
	