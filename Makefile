GROUP=rainbond
VERSION=v1alpha1

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
	./hack/k8s/codegen/update-generated.sh

api-add:
	operator-sdk add api --api-version=rainbond.io/$(VERSION) --kind=$(KIND)

operator-build:
	operator-sdk build abewang/rainbond-operator:v0.0.1

ctrl-add:
	operator-sdk add controller --api-version=rainbond.io/$(VERSION) --kind=$(KIND)