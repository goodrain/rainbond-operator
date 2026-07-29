package controllers

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestApplySystemCriticalDefaultsPreservesConfiguredEphemeralStorage(t *testing.T) {
	t.Parallel()

	deployment := &appsv1.Deployment{
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "component",
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
								},
							},
						},
					},
				},
			},
		},
	}

	applySystemCriticalDefaults(deployment)
	if got := deployment.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceEphemeralStorage]; got.String() != "2Gi" {
		t.Fatalf("expected configured ephemeral-storage request 2Gi, got %s", got.String())
	}
}

func TestApplySystemCriticalDefaultsSupportsAllWorkloadTypes(t *testing.T) {
	t.Parallel()

	deployment := &appsv1.Deployment{Spec: appsv1.DeploymentSpec{Template: podTemplateWithTestContainer()}}
	statefulSet := &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Template: podTemplateWithTestContainer()}}
	daemonSet := &appsv1.DaemonSet{Spec: appsv1.DaemonSetSpec{Template: podTemplateWithTestContainer()}}
	job := &batchv1.Job{Spec: batchv1.JobSpec{Template: podTemplateWithTestContainer()}}
	tests := []struct {
		name     string
		workload client.Object
		podSpec  *corev1.PodSpec
	}{
		{name: "deployment", workload: deployment, podSpec: &deployment.Spec.Template.Spec},
		{name: "statefulset", workload: statefulSet, podSpec: &statefulSet.Spec.Template.Spec},
		{name: "daemonset", workload: daemonSet, podSpec: &daemonSet.Spec.Template.Spec},
		{name: "job", workload: job, podSpec: &job.Spec.Template.Spec},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applySystemCriticalDefaults(tt.workload)
			if got := tt.podSpec.PriorityClassName; got != "system-cluster-critical" {
				t.Fatalf("expected priority class system-cluster-critical, got %q", got)
			}
			if got := tt.podSpec.Containers[0].Resources.Requests[corev1.ResourceEphemeralStorage]; got.String() != "256Mi" {
				t.Fatalf("expected ephemeral-storage request 256Mi, got %s", got.String())
			}
		})
	}
}

func podTemplateWithTestContainer() corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "component"}},
		},
	}
}
