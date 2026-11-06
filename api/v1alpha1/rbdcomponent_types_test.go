package v1alpha1

import (
	"encoding/json"
	"os"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"
)

func TestRbdComponentTolerationsRoundTrip(t *testing.T) {
	input := []byte(`{"spec":{"tolerations":[{"key":"node.kubernetes.io/unschedulable","operator":"Exists","effect":"NoSchedule"},{"key":"dedicated","operator":"Equal","value":"rainbond","effect":"NoExecute","tolerationSeconds":0}]}}`)
	var component RbdComponent
	if err := json.Unmarshal(input, &component); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(component.DeepCopy())
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Spec struct {
			Tolerations []corev1.Toleration `json:"tolerations"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Spec.Tolerations) != 2 {
		t.Fatalf("expected both tolerations to survive API round trip, got %v", result.Spec.Tolerations)
	}
	if got := result.Spec.Tolerations[1]; got.Value != "rainbond" || got.TolerationSeconds == nil || *got.TolerationSeconds != 0 {
		t.Fatalf("expected value and explicit zero tolerationSeconds to survive round trip, got %+v", got)
	}
}

func TestRbdComponentCRDAllowsTolerations(t *testing.T) {
	data, err := os.ReadFile("../../config/crd/bases/rainbond.io_rbdcomponents.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var crd apiextensionsv1.CustomResourceDefinition
	if err := yaml.Unmarshal(data, &crd); err != nil {
		t.Fatal(err)
	}
	for _, version := range crd.Spec.Versions {
		t.Run(version.Name, func(t *testing.T) {
			tolerations := version.Schema.OpenAPIV3Schema.Properties["spec"].Properties["tolerations"]
			if tolerations.Type != "array" || tolerations.Items == nil || tolerations.Items.Schema == nil {
				t.Fatal("CRD must declare spec.tolerations as an array of objects to prevent pruning")
			}
			for _, field := range []string{"key", "operator", "value", "effect", "tolerationSeconds"} {
				if _, exists := tolerations.Items.Schema.Properties[field]; !exists {
					t.Errorf("CRD is missing toleration field %q", field)
				}
			}
		})
	}
}

func TestRbdComponentDeepCopyOwnsTolerations(t *testing.T) {
	seconds := int64(30)
	original := &RbdComponent{Spec: RbdComponentSpec{
		Tolerations: []corev1.Toleration{{Key: "dedicated", TolerationSeconds: &seconds}},
	}}
	clone := original.DeepCopy()
	clone.Spec.Tolerations[0].Key = "changed"
	*clone.Spec.Tolerations[0].TolerationSeconds = 60
	if original.Spec.Tolerations[0].Key != "dedicated" || seconds != 30 {
		t.Fatal("DeepCopy must isolate both the tolerations slice and tolerationSeconds pointers")
	}
}

func TestRegistryCoordinationSchemaAndDeepCopy(t *testing.T) {
	original := &RbdComponent{Spec: RbdComponentSpec{RegistryCoordination: &RegistryCoordinationSpec{Image: "pinned", Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("128Mi")}}}}}
	copied := original.DeepCopy()
	copied.Spec.RegistryCoordination.Image = "changed"
	copied.Spec.RegistryCoordination.Resources.Requests[corev1.ResourceMemory] = resource.MustParse("256Mi")
	if original.Spec.RegistryCoordination.Image != "pinned" || original.Spec.RegistryCoordination.Resources.Requests.Memory().String() != "128Mi" {
		t.Fatal("configuration aliases after DeepCopy")
	}
	data, err := os.ReadFile("../../config/crd/bases/rainbond.io_rbdcomponents.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var crd apiextensionsv1.CustomResourceDefinition
	if err := yaml.Unmarshal(data, &crd); err != nil {
		t.Fatal(err)
	}
	spec := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	config, ok := spec.Properties["registryCoordination"]
	if !ok || config.Type != "object" || len(config.Required) != 10 {
		t.Fatal("missing structural configuration schema")
	}
	for _, name := range spec.Required {
		if name == "registryCoordination" {
			t.Fatal("changed default installation")
		}
	}
}
