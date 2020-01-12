GROUP=rainbond
VERSION=v1alpha1

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

operator-image:
	operator-sdk build abewang/rainbond-operator:v0.0.1

ctrl-add:
	operator-sdk add controller --api-version=rainbond.io/$(VERSION) --kind=$(KIND)

.PHONY: check
check:
	which ./bin/golangci-lint > /dev/null || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s v1.22.2
	@bin/golangci-lint run

test:operator-image
	docker save -o /tmp/rainbond-operator.tgz abewang/rainbond-operator:v0.0.1
	scp /tmp/rainbond-operator.tgz root@172.20.0.12:/root

.PHONY: mock
mock:
	./mockgen.sh

build-api:
	docker build . -f hack/openapi/Dockerfile -t abewang/rbd-operator-openapi