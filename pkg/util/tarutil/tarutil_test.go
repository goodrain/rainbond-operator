package tarutil

import "testing"

func TestUntartar(t *testing.T) {
	err := Untartar("/Users/abewang/Downloads/telecomfive-12-04.zip", "/tmp")
	if err != nil {
		t.Error(err)
	}
}
