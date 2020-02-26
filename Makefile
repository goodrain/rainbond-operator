ifdef IMAGE_NAMESPACE
    IMAGE_NAMESPACE ?= ${IMAGE_NAMESPACE}
else
    IMAGE_NAMESPACE=goodrain
endif

ifdef IMAGE_DOMAIN
    IMAGE_DOMAIN ?= ${IMAGE_DOMAIN}
else
    IMAGE_DOMAIN=registry.cn-hangzhou.aliyuncs.com
endif

ifdef VERSION
	VERSION ?= ${VERSION}
else ifdef TRAVIS_COMMIT
	VERSION ?= v1.0.0-beta2-${TRAVIS_COMMIT}
else 
	VERSION ?= v1.0.0-beta2-$(shell git describe --always --dirty)
endif


GROUP=rainbond
APIVERSION=v1alpha1

PKG             := github.com/goodrain/rainbond-operator
SRC_DIRS        := cmd pkg

.PHONY: test
test:
	@echo "Testing: $(SRC_DIRS)"
	./hack/unit_test
	PKG=$(PKG) ./hack/test $(SRC_DIRS)

.PHONY: build-dirs
build-dirs:
	@echo "Creating build directories"
	@mkdir -p bin/

.PHONY: gen
gen: crds-gen openapi-gen sdk-gen
crds-gen:
	operator-sdk generate crds
openapi-gen:
	# Build the latest openapi-gen from source
	which ./bin/openapi-gen > /dev/null || go build -o ./bin/openapi-gen k8s.io/kube-openapi/cmd/openapi-gen
    # Run openapi-gen for each of your API group/version packages
	./bin/openapi-gen --logtostderr=true \
    -o "" -i ./pkg/apis/$(GROUP)/$(APIVERSION) \
    -O zz_generated.openapi \
    -p ./pkg/apis/$(GROUP)/$(APIVERSION) \
    -h ./hack/k8s/codegen/boilerplate.go.txt -r "-"
sdk-gen:
	chmod +x vendor/k8s.io/code-generator/generate-groups.sh
	./hack/k8s/codegen/update-generated.sh
sdk-verify:
	./hack/k8s/codegen/verify-generated.sh

api-add:
	operator-sdk add api --api-version=rainbond.io/$(APIVERSION) --kind=$(KIND)

ctrl-add:
	operator-sdk add controller --api-version=rainbond.io/$(APIVERSION) --kind=$(KIND)

.PHONY: golangci-lint
golangci-lint: build-dirs
	which ./bin/golangci-lint > /dev/null || sh ./hack/golangci-lint-install.sh v1.23.2
	@bin/golangci-lint run



.PHONY: mock
mock:
	./hack/mockgen.sh

.PHONY: build
build-ui:
	docker build . -f hack/build/ui/Dockerfile -t $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui-base:$(VERSION)
build-api:
	docker build . -f hack/build/openapi/Dockerfile -t $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui:$(VERSION)
build-api-dev:
	docker build . -f hack/build/openapi/Dockerfile.dev -t $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui:$(VERSION)
build-operator:
	docker build . -f hack/build/operator/Dockerfile -t $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rainbond-operator:$(VERSION)
build-operator-dev:
	docker build . -f hack/build/operator/Dockerfile.dev -t $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rainbond-operator:$(VERSION)	
build: build-ui build-api build-operator

docker-login:
	docker login $(IMAGE_DOMAIN) -u $(DOCKER_USER) -p $(DOCKER_PASS)

.PHONY: push
push-ui: build-ui
	docker push $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui-base:$(VERSION)
push-api: build-api
	docker push $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rbd-op-ui:$(VERSION)
push-operator: build-operator
	docker push $(IMAGE_DOMAIN)/$(IMAGE_NAMESPACE)/rainbond-operator:$(VERSION)
push: docker-login push-ui push-api push-operator

chart:
	tar -cvf rainbond-operator-chart.tar ./mychart
	