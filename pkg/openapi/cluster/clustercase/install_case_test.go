package clustercase

import (
	"os"
	"testing"
)

func Test_downloadFile(t *testing.T) {
	if _, err := os.Stat("/tmp/rainbond.tar"); os.IsNotExist(err) {
		t.Log("do not exists, downloading...")
		if err := downloadFile("/tmp/rainbond.tar", "https://github.com/goodrain/rainbond/blob/master/README.md"); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Log("already exists, do not download again")
	}
	t.Log("success")
}
