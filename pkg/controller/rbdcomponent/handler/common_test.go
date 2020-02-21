package handler

import (
	"context"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	rbdutil "github.com/goodrain/rainbond-operator/pkg/util/rbduitl"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestGetDefaultInfo(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	namespace := "rbd-system"
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DBName,
			Namespace: "rbd-system",
		},
		Data: map[string][]byte{
			mysqlPasswordKey: []byte("foobar"),
			mysqlUserKey:     []byte("write"),
		},
	}
	clientset := fake.NewFakeClientWithScheme(scheme, secret)

	dbInfo, err := getDefaultDBInfo(ctx, clientset, nil, namespace, DBName)
	if err != nil {
		t.Errorf("get db info: %v", err)
		t.FailNow()
	}
	assert.NotNil(t, dbInfo)
	assert.Equal(t, "foobar", dbInfo.Password)
	assert.Equal(t, "write", dbInfo.Username)
}

func TestStorageClassRWXVolumeNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	cli := fake.NewFakeClientWithScheme(scheme)
	ctx := context.Background()
	ns := "rbd-system"
	_, err := storageClassNameFromRainbondVolumeRWX(ctx, cli, ns)
	assert.NotNil(t, err)
	assert.True(t, IsRainbondVolumeNotFound(err))
	assert.Equal(t, rainbondVolumeNotFound, err.Error())
}

func TestStorageClassRWXVolumeRWXNotReady(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	ns := "rbd-system"
	labels := rbdutil.LabelsForAccessModeRWX()
	volume := &rainbondv1alpha1.RainbondVolume{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels:    labels,
		},
	}
	cli := fake.NewFakeClientWithScheme(scheme, volume)
	ctx := context.Background()
	_, err := storageClassNameFromRainbondVolumeRWX(ctx, cli, ns)
	assert.NotNil(t, err)
	assert.True(t, IsIgnoreError(err))
	assert.Equal(t, "storage class not ready", err.Error())
}

func TestStorageClassRWXVolumeRWXOK(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	ns := "rbd-system"
	labels := rbdutil.LabelsForAccessModeRWX()
	sc := "foobar.csi.rainbond.io"
	volume := &rainbondv1alpha1.RainbondVolume{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels:    labels,
		},
		Spec: rainbondv1alpha1.RainbondVolumeSpec{
			StorageClassName: sc,
		},
	}
	cli := fake.NewFakeClientWithScheme(scheme, volume)
	ctx := context.Background()
	got, err := storageClassNameFromRainbondVolumeRWX(ctx, cli, ns)
	assert.Nil(t, err)
	assert.Equal(t, sc, got)
}

func TestStorageClassRWXVolumeRWONotFoundAndRWXNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	ns := "rbd-system"
	cli := fake.NewFakeClientWithScheme(scheme)
	ctx := context.Background()
	_, err := storageClassNameFromRainbondVolumeRWO(ctx, cli, ns)
	assert.NotNil(t, err)
	assert.True(t, IsRainbondVolumeNotFound(err))
	assert.Equal(t, rainbondVolumeNotFound, err.Error())
}

func TestStorageClassRWXVolumeRWONotFoundButRWXFound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	ns := "rbd-system"
	labels := rbdutil.LabelsForAccessModeRWX()
	sc := "foobar.csi.rainbond.io"
	volume := &rainbondv1alpha1.RainbondVolume{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Labels:    labels,
		},
		Spec: rainbondv1alpha1.RainbondVolumeSpec{
			StorageClassName: sc,
		},
	}
	cli := fake.NewFakeClientWithScheme(scheme, volume)
	ctx := context.Background()
	got, err := storageClassNameFromRainbondVolumeRWO(ctx, cli, ns)
	assert.Nil(t, err)
	assert.Equal(t, sc, got)
}

func TestStorageClassRWXVolumeRWOOK(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	ns := "rbd-system"

	volumerwo := getVolume(ns, rbdutil.LabelsForAccessModeRWO())
	volumerwx := getVolume(ns, rbdutil.LabelsForAccessModeRWX())

	cli := fake.NewFakeClientWithScheme(scheme, volumerwo, volumerwx)
	ctx := context.Background()
	got, err := storageClassNameFromRainbondVolumeRWO(ctx, cli, ns)
	assert.Nil(t, err)
	assert.Equal(t, volumerwo.Spec.StorageClassName, got)
}

func TestSetStorageCassNameRWX(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	ns := "rbd-system"

	volumerwx := getVolume(ns, rbdutil.LabelsForAccessModeRWX())
	cli := fake.NewFakeClientWithScheme(scheme, volumerwx)
	ctx := context.Background()

	dummyStorageClassRWX := &dummyStorageClassRWX{}
	err := setStorageCassName(ctx, cli, ns, dummyStorageClassRWX)
	assert.Nil(t, err)
	assert.Equal(t, volumerwx.Spec.StorageClassName, dummyStorageClassRWX.storageClassRWX)
}

func TestSetStorageCassNameRWO(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	ns := "rbd-system"

	volumerwo := getVolume(ns, rbdutil.LabelsForAccessModeRWO())
	cli := fake.NewFakeClientWithScheme(scheme, volumerwo)
	ctx := context.Background()

	dummyStorageClassRWO := &dummyStorageClassRWO{}
	err := setStorageCassName(ctx, cli, ns, dummyStorageClassRWO)
	assert.Nil(t, err)
	assert.Equal(t, volumerwo.Spec.StorageClassName, dummyStorageClassRWO.storageClassRWO)
}

func TestSetStorageCassNameBothRWXAndRWO(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	ns := "rbd-system"

	volumerwx := getVolume(ns, rbdutil.LabelsForAccessModeRWX())
	volumerwo := getVolume(ns, rbdutil.LabelsForAccessModeRWO())
	cli := fake.NewFakeClientWithScheme(scheme, volumerwx, volumerwo)
	ctx := context.Background()

	dummyStorageClass := &dummyStorageClass{}
	err := setStorageCassName(ctx, cli, ns, dummyStorageClass)
	assert.Nil(t, err)
	assert.Equal(t, volumerwo.Spec.StorageClassName, dummyStorageClass.storageClassRWO)
	assert.Equal(t, volumerwx.Spec.StorageClassName, dummyStorageClass.storageClassRWX)
}

func getVolume(ns string, labels map[string]string) *rainbondv1alpha1.RainbondVolume {
	sc := "foo" + labels["accessModes"] + ".csi.rainbond.io"
	volume := &rainbondv1alpha1.RainbondVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      labels["accessModes"],
			Namespace: ns,
			Labels:    labels,
		},
		Spec: rainbondv1alpha1.RainbondVolumeSpec{
			StorageClassName: sc,
		},
	}
	return volume
}

type dummyStorageClassRWX struct {
	storageClassRWX string
}

func (d *dummyStorageClassRWX) SetStorageClassNameRWX(sc string) {
	d.storageClassRWX = sc
}

type dummyStorageClassRWO struct {
	storageClassRWO string
}

func (d *dummyStorageClassRWO) SetStorageClassNameRWO(sc string) {
	d.storageClassRWO = sc
}

type dummyStorageClass struct {
	dummyStorageClassRWX
	dummyStorageClassRWO
}
