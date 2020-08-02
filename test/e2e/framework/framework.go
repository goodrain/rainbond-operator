package framework

import (
	rainbondoperator "github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// Framework supports common operations used by e2e tests; it will keep a kubernetes cluster for you.
type Framework struct {
	BaseName string

	// A Kubernetes and Service Catalog client
	KubeClientSet     kubernetes.Interface
	KubeConfig        *restclient.Config
	RainbondClientset rainbondoperator.Interface

	Namespace    string
	OperatorName string
}

// NewDefaultFramework makes a new framework and sets up a BeforeEach/AfterEach for
// you (you can write additional before/after each functions).
func NewDefaultFramework(baseName string) *Framework {
	f := &Framework{
		BaseName:     baseName,
		OperatorName: "rainbond-operator",
	}

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

// BeforeEach gets a client and makes a namespace.
func (f *Framework) BeforeEach() {
	kubeConfig := k8sutil.MustNewKubeConfig("/Users/abewang/.kube/config") // TODO: do not hard code
	f.KubeConfig = kubeConfig

	By("Creating a kubernetes client")
	var err error
	f.KubeClientSet, err = kubernetes.NewForConfig(kubeConfig)
	Expect(err).NotTo(HaveOccurred())

	By("Creating a rainbond operator client")
	f.RainbondClientset, err = rainbondoperator.NewForConfig(kubeConfig)
	Expect(err).NotTo(HaveOccurred())

	By("Building a namespace api object")
	namespace, err := CreateKubeNamespace(f.BaseName, f.KubeClientSet)
	Expect(err).NotTo(HaveOccurred())
	f.Namespace = namespace

	By("Creating a new rainbond operator")
	err = InstallReainbondOperator(f.OperatorName, "/Users/abewang/go/src/github.com/goodrain/rainbond-operator/chart", namespace)
	Expect(err).NotTo(HaveOccurred())

	err = WaitForPodsReady(f.KubeClientSet, DefaultTimeout, 1, f.Namespace, metav1.ListOptions{
		LabelSelector: "name=rainbond-operator",
	})
	Expect(err).NotTo(HaveOccurred())
}

// AfterEach deletes the namespace, after reading its events.
func (f *Framework) AfterEach() {
	By("Delete ClusterRoleBinding")
	err := DeleteClusterRoleBinding(f.KubeClientSet, f.OperatorName)
	Expect(err).NotTo(HaveOccurred())

	By("Waiting for test namespace to no longer exist")
	err = DeleteKubeNamespace(f.KubeClientSet, f.Namespace)
	Expect(err).NotTo(HaveOccurred())
}
