package nodestore

import (
	"context"
	"testing"

	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"gopkg.in/stretchr/testify.v1/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSearchbyIIP(t *testing.T) {
	testdata := []*v1.K8sNode{
		{
			Name:       "apple",
			InternalIP: "192.168.0.1",
		},
		{
			Name:       "banana",
			InternalIP: "192.168.0.2",
		},
		{
			Name:       "cat",
			InternalIP: "172.20.0.20",
		},
	}

	tests := []struct {
		name, query    string
		k8sNodes, want []*v1.K8sNode
	}{
		{
			name:     "ok",
			query:    "192.168.0.1",
			k8sNodes: testdata,
			want: []*v1.K8sNode{
				{
					Name:       "apple",
					InternalIP: "192.168.0.1",
				},
			},
		},
		{
			name:     "prefix",
			query:    "192.168",
			k8sNodes: testdata,
			want: []*v1.K8sNode{
				{
					Name:       "apple",
					InternalIP: "192.168.0.1",
				},
				{
					Name:       "banana",
					InternalIP: "192.168.0.2",
				},
			},
		},
		{
			name:     "match nothing",
			query:    "39.",
			k8sNodes: testdata,
		},
	}
	for idx := range tests {
		tc := tests[idx]
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			nodeList := generateNodeList(tc.k8sNodes)
			clientset := fake.NewSimpleClientset(nodeList)

			store := New(ctx, clientset)
			if err := store.Start(); err != nil {
				t.Errorf("start node store: %v", err)
				t.FailNow()
			}

			nodes := store.SearchByIIP(tc.query, false)
			assert.EqualValues(t, tc.want, nodes)
		})
	}

}

func generateNodeList(k8sNodes []*v1.K8sNode) *corev1.NodeList {
	var items []corev1.Node
	for _, node := range k8sNodes {
		item := corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: node.Name,
			},
			Status: corev1.NodeStatus{
				Addresses: []corev1.NodeAddress{
					{
						Type:    corev1.NodeInternalIP,
						Address: node.InternalIP,
					},
				},
			},
		}
		items = append(items, item)
	}

	return &corev1.NodeList{
		Items: items,
	}
}
