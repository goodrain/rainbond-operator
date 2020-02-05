package downloadutil

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestDownloadWithProgress_Download(t *testing.T) {
	fmt.Println("start")
	dp := DownloadWithProgress{URL: "http://localhost/fanyangyang/Downloads/rainbond-pkg-V5.2-dev.tgz", SavedPath: "/opt/rainbond/pkg/rainbond-pkg-V5.2-dev.tgz", Wanted: "d41d8cd98f00b204e9800998ecf8427e"}
	go func() {
		for {
			fmt.Println("precent : ", dp.Percent)
			time.Sleep(time.Second / 2)
		}
	}()
	if err := dp.Download(); err != nil {
		t.Fatal(err)
	}
	t.Log("success")
}

func TestMd5(t *testing.T) {
	dp := DownloadWithProgress{Wanted: "c82f8782ee1b71443799ca6182d017ea"}
	target, err := os.Open("/opt/rainbond/pkg/rainbond-pkg-V5.2-dev.tgz")
	if err != nil {
		t.Fatal(err)
	}
	if err := dp.checkMD5(target); err != nil {
		t.Fail()
	}
}
