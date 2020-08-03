package suffixdomain

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/uuid"
)

func TestGenerateDomain(t *testing.T) {
	re, err := GenerateDomain("192.168.2.2", string(uuid.NewUUID()), string(uuid.NewUUID()))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(re)
}
