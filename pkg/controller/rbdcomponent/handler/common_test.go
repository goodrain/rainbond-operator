package handler

import (
	"context"
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
