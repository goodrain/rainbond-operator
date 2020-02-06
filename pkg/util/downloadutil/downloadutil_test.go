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
	if err := dp.CheckMD5(target); err != nil {
		t.Fail()
	}
}

func TestSha256Sum(t *testing.T) {
	//target, err := os.Open("/Users/fanyangyang/Downloads/go1.13.7.src.tar.gz")
	//target, err := os.Open("/Users/fanyangyang/Downloads/rainbond.images.2020-02-05-5.2-dev.tgz")
	target, err := os.Open("/opt/rainbond/pkg/rainbond-pkg.tar")
	if err != nil {
		t.Fatal(err)
	}
	dp := DownloadWithProgress{Wanted: "fcd61975ff0a55fc1a1dd997043488adc14fe7e4fea474f77865a0689b52e1de"}
	t.Log(dp.CheckMD5(target))

}
