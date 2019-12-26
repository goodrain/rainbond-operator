package privateregistry

import (
	"testing"
	"fmt"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	"github.com/ghodss/yaml"
	"github.com/davecgh/go-spew/spew"
)

func TestPersistentVolumeClaimForPrivateRegistry(t *testing.T){
	pr := &rainbondv1alpha1.PrivateRegistry{}
	pvc := persistentVolumeClaimForPrivateRegistry(pr)
	y, _ := toYAML(pvc)
	t.Logf("%v\n", y)
}

func toYAML(v interface{}) (string, error) {
	y, err := yaml.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("yaml marshal failed:%v\n%v\n", err, spew.Sdump(v))
	}

	return string(y), nil
}