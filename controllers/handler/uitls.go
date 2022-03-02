package handler

import (
	"fmt"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func nodeAffnityNodesForChaos(cluster *wutongv1alpha1.WutongCluster) (*corev1.VolumeNodeAffinity, error) {
	if len(cluster.Spec.NodesForChaos) == 0 {
		// TODO: Is it neccessary to check NodesForChaos?
		return nil, fmt.Errorf("nodes for chaos not found")
	}

	return &corev1.VolumeNodeAffinity{
		Required: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/hostname",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{cluster.Spec.NodesForChaos[0].Name},
						},
					},
				},
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "k3s.io/hostname",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{cluster.Spec.NodesForChaos[0].Name},
						},
					},
				},
			},
		},
	}, nil
}
