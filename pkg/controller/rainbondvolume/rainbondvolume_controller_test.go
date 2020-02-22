package rainbondvolume

import (
	"context"
	"github.com/golang/mock/gomock"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/controller/rainbondvolume/plugin/mock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	cli := fake.NewFakeClientWithScheme(scheme, volume)
	r := &ReconcileRainbondVolume{client: cli}

	request := reconcile.Request{}
	request.Name = name
	result, err := r.Reconcile(request)
	assert.Equal(t, reconcile.Result{}, result)
	assert.Equal(t, nil, err)

	ctx := context.Background()
	newVolume := &rainbondv1alpha1.RainbondVolume{}
	if err := cli.Get(ctx, types.NamespacedName{Name: name}, newVolume); err != nil {
		t.Errorf("rainbondvolume: %v", err)
		t.FailNow()
	}
	_, condition := newVolume.Status.GetRainbondVolumeCondition(rainbondv1alpha1.RainbondVolumeReady)
	assert.NotNil(t, condition)
	assert.Equal(t, condition.Status, corev1.ConditionTrue)
}

func TestReconcileStorageClassParameterNotNil(t *testing.T) {
	name := "rainbondvolume"

	tests := []struct {
		name   string
		volume *rainbondv1alpha1.RainbondVolume
		class  *storagev1.StorageClass
		want   *storagev1.StorageClass
	}{
		{
			name: "storageclass found",
			volume: &rainbondv1alpha1.RainbondVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: rainbondv1alpha1.RainbondVolumeSpec{
					StorageClassParameters: &rainbondv1alpha1.StorageClassParameters{
						Provisioner: "foobar",
					},
				},
			},
			class: &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			},
			want: &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			},
		},
		{
			name: "storageclass not found",
			volume: &rainbondv1alpha1.RainbondVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec: rainbondv1alpha1.RainbondVolumeSpec{
					StorageClassParameters: &rainbondv1alpha1.StorageClassParameters{
						Provisioner: "rainbond.io/abc",
					},
				},
			},
			class: &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foobar",
				},
			},
			want: &storagev1.StorageClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
			},
		},
	}
	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)
	_ = storagev1.AddToScheme(scheme)

	ctx := context.Background()

	for idx := range tests {
		tc := tests[idx]
		t.Run(tc.name, func(t *testing.T) {
			var cli client.Client
			if tc.class != nil {
				cli = fake.NewFakeClientWithScheme(scheme, tc.volume, tc.class)
			} else {
				cli = fake.NewFakeClientWithScheme(scheme, tc.volume)
			}

			r := &ReconcileRainbondVolume{client: cli}

			request := reconcile.Request{}
			request.Name = name
			_, err := r.Reconcile(request)
			assert.Nil(t, err)

			sc := &storagev1.StorageClass{}
			if err = r.client.Get(ctx, types.NamespacedName{Name: name}, sc); err != nil {
				t.Errorf("get storageclass: %v", err)
				t.FailNow()
			}
			assert.EqualValues(t, tc.want.Name, sc.Name)

			newVolume := &rainbondv1alpha1.RainbondVolume{}
			if err := cli.Get(ctx, types.NamespacedName{Name: name}, newVolume); err != nil {
				t.Errorf("rainbondvolume: %v", err)
				t.FailNow()
			}
			assert.NotNil(t, newVolume)
			assert.Equal(t, name, newVolume.Spec.StorageClassName)
			_, condition := newVolume.Status.GetRainbondVolumeCondition(rainbondv1alpha1.RainbondVolumeReady)
			assert.NotNil(t, condition)
			assert.Equal(t, condition.Status, corev1.ConditionTrue)
		})
	}
}

func TestApplyCSIPluginCSIDriverExists(t *testing.T) {
	provisioenr := "foobar.csi.rainbond.io"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	plugin := mock.NewMockCSIPlugin(ctrl)
	plugin.EXPECT().CheckIfCSIDriverExists().Return(true)
	plugin.EXPECT().GetProvisioner().Return(provisioenr)

	volume := &rainbondv1alpha1.RainbondVolume{}

	ctx := context.Background()

	r := &ReconcileRainbondVolume{}
	err := r.applyCSIPlugin(ctx, plugin, volume)
	assert.Nil(t, err)
	assert.Equal(t, provisioenr, volume.Spec.StorageClassParameters.Provisioner)
}
