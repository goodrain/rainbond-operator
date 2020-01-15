package downloadutil

import (
	"fmt"
	"testing"
	"time"
)

func TestDownloadWithProgress_Download(t *testing.T) {
	fmt.Println("start")
	dp := DownloadWithProgress{URL: "http://localhost/fanyangyang/Downloads/rainbond-pkg-V5.2-dev.tgz", SavedPath: "/opt/rainbond/pkg/rainbond-pkg-V5.2-dev.tgz"}
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
