package tarutil

import "testing"

func TestUntartar(t *testing.T) {
	err := Untartar("/root/Downloads/rainbond.images.2020-02-07-5.2-dev.tgz", "/tmp")
	if err != nil {
		t.Error(err)
	}
}
