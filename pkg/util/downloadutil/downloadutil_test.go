package downloadutil

import (
	"fmt"
	"testing"
	"time"
)

func TestDownloadWithProgress_Download(t *testing.T) {
	dp := DownloadWithProgress{URL: "https://github.com/schollz/croc/releases/download/v4.1.4/croc_v4.1.4_Windows-64bit_GUI.zip", SavedPath: "/opt/rainbond/pkg/rainbond-pkg-V5.2-dev.tgz"}
	go func() {
		for {
			fmt.Println("precent : ", dp.Percent)
			time.Sleep(time.Second * 2)
		}
	}()
	if err := dp.Download(); err != nil {
		t.Fatal(err)
	}
	t.Log("success")
}
