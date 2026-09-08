package handler

import (
	"reflect"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func TestHostsAliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		imageHub    *rainbondv1alpha1.ImageHub
		gatewayIP   string
		wantAliases []corev1.HostAlias
	}{
		{
			name:      "default repository",
			imageHub:  &rainbondv1alpha1.ImageHub{Domain: "goodrain.me"},
			gatewayIP: "10.0.0.10",
			wantAliases: []corev1.HostAlias{
				{IP: "10.0.0.10", Hostnames: []string{"goodrain.me"}},
			},
		},
		{
			name: "default repository with namespace",
			imageHub: &rainbondv1alpha1.ImageHub{
				Domain:    "goodrain.me",
				Namespace: "rainbond",
			},
			gatewayIP: "10.0.0.10",
			wantAliases: []corev1.HostAlias{
				{IP: "10.0.0.10", Hostnames: []string{"goodrain.me"}},
			},
		},
		{
			name:      "default repository with port",
			imageHub:  &rainbondv1alpha1.ImageHub{Domain: "goodrain.me:9443"},
			gatewayIP: "10.0.0.10",
			wantAliases: []corev1.HostAlias{
				{IP: "10.0.0.10", Hostnames: []string{"goodrain.me"}},
			},
		},
		{
			name:        "custom repository",
			imageHub:    &rainbondv1alpha1.ImageHub{Domain: "registry.example.com"},
			gatewayIP:   "10.0.0.10",
			wantAliases: nil,
		},
		{
			name:        "missing gateway IP",
			imageHub:    &rainbondv1alpha1.ImageHub{Domain: "goodrain.me"},
			wantAliases: nil,
		},
	}

	for _, tt := range tests {
		current := tt
		t.Run(current.name, func(t *testing.T) {
			t.Parallel()

			cluster := &rainbondv1alpha1.RainbondCluster{
				Spec: rainbondv1alpha1.RainbondClusterSpec{
					ImageHub: current.imageHub,
				},
			}
			if current.gatewayIP != "" {
				cluster.Spec.NodesForGateway = []*rainbondv1alpha1.K8sNode{
					{InternalIP: current.gatewayIP},
				}
			}

			if got := hostsAliases(cluster); !reflect.DeepEqual(got, current.wantAliases) {
				t.Fatalf("hostsAliases() = %v, want %v", got, current.wantAliases)
			}
		})
	}
}
