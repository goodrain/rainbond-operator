package handler

import (
	"context"
	"errors"
	"fmt"
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrNoDBEndpoints = errors.New("no ready endpoints for DB were found")

func isDBReady(ctx context.Context, cli client.Client) error {
	eps := &corev1.EndpointsList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"name":     DBName,
			"belongTo": "RainbondOperator", // TODO: DO NOT HARD CODE
		}),
	}
	if err := cli.List(ctx, eps, listOpts...); err != nil {
		return err
	}
	for _, ep := range eps.Items {
		for _, subset := range ep.Subsets {
			if len(subset.Addresses) > 0 {
				return nil
			}
		}
	}
	return ErrNoDBEndpoints
}

func getDefaultDBInfo(in *rainbondv1alpha1.Database) *rainbondv1alpha1.Database {
	if in != nil {
		return in
	}
	return &rainbondv1alpha1.Database{
		Host:     DBName,
		Port:     3306,
		Username: "root",
		Password: "rainbond",
	}
}

func isPhaseOK(cluster *rainbondv1alpha1.RainbondCluster) error {
	if cluster.Spec.InstallMode == rainbondv1alpha1.InstallationModeWithoutPackage {
		return nil
	}

	pkgOK := rainbondv1alpha1.RainbondClusterPhase2Range[cluster.Status.Phase] > rainbondv1alpha1.RainbondClusterPhase2Range[rainbondv1alpha1.RainbondClusterPackageProcessing]
	if cluster.Status == nil || !pkgOK {
		return fmt.Errorf("rainbond package processing")
	}

	return nil
}
