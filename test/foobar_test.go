package test

import (
	"testing"

	"github.com/GLYASAI/rainbond-operator/pkg/generated/clientset/versioned"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func TestFoobar(t *testing.T) {
	c, err := clientcmd.BuildConfigFromFlags("", "./admin.conf")
	if err != nil {
		t.Fatalf("error reading kube config file: %s", err.Error())
	}

	clientset, err := versioned.NewForConfig(c)
	if err != nil {
		t.Fatalf("create clientset: %v", err)
	}

	rainbond, err := clientset.PrivateregistryV1alpha1().PrivateRegistries("default").Get("foobar", metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		t.Error(err)
	}
	t.Log(rainbond)
}
