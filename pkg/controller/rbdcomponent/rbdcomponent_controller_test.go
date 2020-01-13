package rbdcomponent

import (
	"github.com/GLYASAI/rainbond-operator/pkg/controller/rbdcomponent/handler"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
)

func TestDetectControllerType(t *testing.T) {
	tests := []struct {
		name          string
		rbdcomponent  *rainbondv1alpha1.RbdComponent
		resourcesFunc resourcesFunc
		want          rainbondv1alpha1.ControllerType
	}{
		{
			name:          "deployment",
			rbdcomponent:  &rainbondv1alpha1.RbdComponent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rbd-app-ui",
					Namespace: "rbd-system",
				},
			},
			resourcesFunc: handler.resourcesForAppUI,
			want:          rainbondv1alpha1.ControllerTypeDeployment,
		},
		{
			name:          "daemonset",
			rbdcomponent:  &rainbondv1alpha1.RbdComponent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rbd-api",
					Namespace: "rbd-system",
				},
			},
			resourcesFunc: handler.resourcesForAPI,
			want:          rainbondv1alpha1.ControllerTypeDaemonSet,
		},
	}

	for idx := range tests {
		tc := tests[idx]
		t.Run(tc.name, func(t *testing.T) {
			got := rainbondv1alpha1.ControllerTypeUnknown

			for _, res := range tc.resourcesFunc(tc.rbdcomponent) {
				if ct := detectControllerType(res); ct != rainbondv1alpha1.ControllerTypeUnknown {
					got = ct
				}
			}

			if got != tc.want {
				t.Errorf("Expected %s, but returned %s", tc.want, got)
			}
		})
	}
}
