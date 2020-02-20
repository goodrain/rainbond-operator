package rainbondvolume

import (
	"context"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

func TestReconcileHaveStorageClassName(t *testing.T) {
	name := "rainbondvolume"

	volume := &rainbondv1alpha1.RainbondVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: rainbondv1alpha1.RainbondVolumeSpec{
			StorageClassName: "foo",
		},
	}
	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)
	client := fake.NewFakeClientWithScheme(scheme, volume)
	r := &ReconcileRainbondVolume{client: client}

	request := reconcile.Request{}
	request.Name = name
	result, err := r.Reconcile(request)
	assert.Equal(t, reconcile.Result{}, result)
	assert.Equal(t, nil, err)

	ctx := context.Background()
	newVolume := &rainbondv1alpha1.RainbondVolume{}
	if err := client.Get(ctx, types.NamespacedName{Name: name}, newVolume); err != nil {
		t.Errorf("rainbondvolume: %v", err)
		t.FailNow()
	}
	_, condition := newVolume.Status.GetRainbondVolumeCondition(rainbondv1alpha1.RainbondVolumeReady)
	assert.NotNil(t, condition)
	assert.Equal(t, condition.Status, corev1.ConditionTrue)
}
