package handler

import (
	"fmt"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func nodeAffnityNodesForChaos(cluster *rainbondv1alpha1.RainbondCluster) (*corev1.VolumeNodeAffinity, error) {
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
			},
		},
	}, nil
}
