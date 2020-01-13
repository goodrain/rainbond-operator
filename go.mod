module github.com/GLYASAI/rainbond-operator

go 1.13

require (
	github.com/aliyun/aliyun-oss-go-sdk v2.0.5+incompatible
	github.com/baiyubin/aliyun-sts-go-sdk v0.0.0-20180326062324-cfa1a18b161f // indirect
	github.com/cheggaaa/pb v2.0.7+incompatible
	github.com/cheggaaa/pb/v3 v3.0.3
	github.com/coreos/etcd v3.3.15+incompatible
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/docker/docker v0.7.3-0.20190817195342-4760db040282
	github.com/gin-contrib/cors v1.3.0 // indirect
	github.com/gin-gonic/gin v1.5.0
	github.com/go-logr/logr v0.1.0
	github.com/golang/mock v1.3.1
	github.com/google/go-cmp v0.3.2-0.20191028172631-481baca67f93 // indirect
	github.com/jinzhu/gorm v1.9.11
	github.com/operator-framework/operator-sdk v0.13.0
	github.com/pquerna/ffjson v0.0.0-20180717144149-af8b230fcd20
	github.com/schollz/progressbar/v2 v2.15.0
	github.com/sirupsen/logrus v1.4.2
	github.com/sparrc/go-ping v0.0.0-20190613174326-4e5b6552494c // indirect
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.4.0
	gopkg.in/VividCortex/ewma.v1 v1.1.1 // indirect
	gopkg.in/cheggaaa/pb.v2 v2.0.7 // indirect
	gopkg.in/fatih/color.v1 v1.7.0 // indirect
	gopkg.in/mattn/go-colorable.v0 v0.1.0 // indirect
	gopkg.in/mattn/go-isatty.v0 v0.0.4 // indirect
	gopkg.in/mattn/go-runewidth.v0 v0.0.4 // indirect
	k8s.io/api v0.0.0
	k8s.io/apimachinery v0.0.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.0.0
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20190918143330-0270cf2f1c1d
	sigs.k8s.io/controller-runtime v0.4.0
)

// Pinned to kubernetes-1.16.2
replace (
	// github.com/docker/docker => github.com/docker/docker v1.4.2-0.20170504205632-89658bed64c2
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191016114015-74ad18325ed5
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191016115326-20453efc2458
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191016111319-039242c015a9
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191016115521-756ffa5af0bd
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016114939-2b2b218dc1df
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016114407-2e83b6f20229
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016114748-65049c67a58b
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191016120415-2ed914427d51
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016114556-7841ed97f1b2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191016115753-cf0698c3a16b
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191016113814-3b1a734dba6e
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191016112829-06bb3c9d77c9
)
