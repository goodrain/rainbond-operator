package k8sutil

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func UpdateOrCreateResource(reqLogger logr.Logger, cli client.Client, obj runtime.Object, meta metav1.Object) (reconcile.Result, error) {
	err := cli.Get(context.TODO(), types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, obj)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new", obj.GetObjectKind().GroupVersionKind().Kind, "Namespace", meta.GetNamespace(), "Name", meta.GetName())
		err = cli.Create(context.TODO(), obj)
		if err != nil {
			reqLogger.Error(err, "Failed to create new", obj.GetObjectKind(), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
			return reconcile.Result{}, err
		}
		// daemonset created successfully - return and requeue TODO: why?
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get ", obj.GetObjectKind())
		return reconcile.Result{}, err
	}

	// obj exsits, update
	reqLogger.Info("Update ", obj.GetObjectKind().GroupVersionKind().Kind, "Namespace", meta.GetNamespace(), "Name", meta.GetName())
	if err := cli.Update(context.TODO(), obj); err != nil {
		reqLogger.Error(err, "Failed to update ", obj.GetObjectKind())
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
