package framework

import (
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
	KubeClientSet kubernetes.Interface
	KubeConfig    *restclient.Config

	Namespace string
}

// NewDefaultFramework makes a new framework and sets up a BeforeEach/AfterEach for
// you (you can write additional before/after each functions).
func NewDefaultFramework(baseName string) *Framework {
	f := &Framework{
		BaseName: baseName,
	}

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f
}

// BeforeEach gets a client and makes a namespace.
func (f *Framework) BeforeEach() {
	By("Creating a kubernetes client")
	kubeConfig, err := restclient.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	Expect(err).NotTo(HaveOccurred())

	f.KubeConfig = kubeConfig
	f.KubeClientSet, err = kubernetes.NewForConfig(kubeConfig)
	Expect(err).NotTo(HaveOccurred())

	By("Building a namespace api object")
	namespace, err := CreateKubeNamespace(f.BaseName, f.KubeClientSet)
	Expect(err).NotTo(HaveOccurred())

	f.Namespace = namespace

	err = WaitForPodsReady(f.KubeClientSet, DefaultTimeout, 1, f.Namespace, metav1.ListOptions{
		LabelSelector: "name=rainbond-operator",
	})
	Expect(err).NotTo(HaveOccurred())
}

// AfterEach deletes the namespace, after reading its events.
func (f *Framework) AfterEach() {
	By("Waiting for test namespace to no longer exist")
	err := DeleteKubeNamespace(f.KubeClientSet, f.Namespace)
	Expect(err).NotTo(HaveOccurred())
}
