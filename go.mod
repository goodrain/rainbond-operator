module github.com/goodrain/rainbond-operator

go 1.15

require (
	github.com/Microsoft/hcsshim v0.9.4 // indirect
	github.com/aliyun/aliyun-oss-go-sdk v2.1.5+incompatible
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/containerd/containerd v1.5.7
	github.com/containerd/continuity v0.3.0 // indirect
	github.com/coreos/etcd v3.3.13+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.2+incompatible
	github.com/gin-gonic/gin v1.6.3
	github.com/go-logr/logr v0.3.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/golang/mock v1.6.0
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/testing v0.0.0-20201216035041-2be42bba85f3 // indirect
	github.com/klauspost/compress v1.11.13
	github.com/myesui/uuid v1.0.0 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.3
	github.com/opencontainers/runc v1.1.4 // indirect
	github.com/pquerna/ffjson v0.0.0-20190930134022-aa0246cd15f7
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.6.1
	github.com/twinj/uuid v1.0.0
	gopkg.in/stretchr/testify.v1 v1.2.2 // indirect
	k8s.io/api v0.20.6
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	k8s.io/kube-aggregator v0.20.1
	sigs.k8s.io/controller-runtime v0.7.0
)

replace google.golang.org/grpc => google.golang.org/grpc v1.29.0
