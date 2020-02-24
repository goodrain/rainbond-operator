package usecase

import (
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned/fake"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestClusterUsecase_StatusInfo404(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)
	rainbondClient := fake.NewSimpleClientset()

	clusterUsecase := &clusterUsecase{namespace: "foo", rainbondClient: rainbondClient, clusterName: "bar"}
	_, err := clusterUsecase.StatusInfo()
	assert.Equal(t, bcode.NotFound, err)
}
