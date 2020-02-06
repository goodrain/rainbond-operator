package usecase

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/cheggaaa/pb"
	pbv3 "github.com/cheggaaa/pb/v3"
	"github.com/gin-gonic/gin"
	"github.com/schollz/progressbar/v2"
)

func Test_downloadFile(t *testing.T) {
	// if _, err := os.Stat("/tmp/rainbond.tar"); os.IsNotExist(err) {
	// 	t.Log("do not exists, downloading...")
	// 	ic := InstallUseCaseImpl{}
	// 	if err := ic.downloadFile(); err != nil {
	// 		t.Fatal(err)
	// 	}
	// } else {
	// 	t.Log("already exists, do not download again")
	// }
	t.Log("success")
}

func Test1(t *testing.T) {
	var limit int64 = 1024 * 1024 * 500
	// we will copy 200 Mb from /dev/rand to /dev/null
	reader := io.LimitReader(rand.Reader, limit-1000)
	writer := ioutil.Discard

	// start bar based on our template
	bar := pb.Full.Start64(limit)
	// create proxy reader
	barReader := bar.NewProxyReader(reader)
	// copy from proxy reader
	_, _ = io.Copy(writer, barReader)
	// finish bar
	bar.Finish()
}

func Test2(t *testing.T) {
	resp, err := http.Get("https://github.com/schollz/croc/releases/download/v4.1.4/croc_v4.1.4_Windows-64bit_GUI.zip")
	if err != nil { // TODO fanyangyang if can't create connection, download manual and upload it
		t.Fatal("get error : ", err.Error())
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create("/tmp/rainbond.tar") // TODO fanyangyang file path and generate test case
	if err != nil {
		t.Fatal("create errror: ", err.Error())
	}
	defer out.Close()
	// start new bar

	bar := pbv3.Full.Start64(resp.ContentLength)
	state := pbv3.State{ProgressBar: bar}
	go func() {
		for {
			t.Log("-----------------", state.Current())
			t.Log("-----", state.IsFinished())
			time.Sleep(time.Second / 2)
		}
	}()
	// create proxy reader
	barReader := bar.NewProxyReader(resp.Body)
	// Write the body to file
	_, _ = io.Copy(out, barReader)

	// t.Log("", bar.State().IsFinished())

	if err := os.Remove("/tmp/rainbond.tar"); err != nil {
		t.Fatal(err)
	}
}

func Test3(t *testing.T) {
	urlToGet := "https://github.com/schollz/croc/releases/download/v4.1.4/croc_v4.1.4_Windows-64bit_GUI.zip"
	req, _ := http.NewRequest("GET", urlToGet, nil)
	resp, _ := http.DefaultClient.Do(req)
	defer func() { _ = resp.Body.Close() }()

	var out io.Writer
	f, _ := os.OpenFile("croc_v4.1.4_Windows-64bit_GUI.zip", os.O_CREATE|os.O_WRONLY, 0644)
	out = f
	defer func() { _ = f.Close() }()

	bar := progressbar.NewOptions(
		int(resp.ContentLength),
		progressbar.OptionSetBytes(int(resp.ContentLength)),
	)
	go func() {
		for {
			t.Logf("-----------------pencent: %v, bytes: %v, all: %v, finisih: %v", bar.State().CurrentPercent, bar.State().CurrentBytes, bar.State().MaxBytes, (bar.State().MaxBytes-int64(bar.State().CurrentBytes)) < 1)
			time.Sleep(time.Second / 2)
		}
	}()
	_ = io.MultiWriter(out, bar)

	//io.Copy(out, resp.Body)
	t.Logf("-----------------pencent: %v, bytes: %v, all: %v, finisih: %v", bar.State().CurrentPercent, bar.State().CurrentBytes, bar.State().MaxBytes, (bar.State().MaxBytes-int64(bar.State().CurrentBytes)) < 1)
}

func Test4(t *testing.T) {
	source := &v1alpha1.RainbondCluster{Status: &v1alpha1.RainbondClusterStatus{}}
	if source.Status.NodeAvailPorts != nil {
		for _, node := range source.Status.NodeAvailPorts {
			t.Logf("%+v", node)
		}
	}
	for _, node := range source.Status.NodeAvailPorts {
		t.Logf("%+v", node)
	}

	var ss1 []string
	ss1 = append(ss1, source.Spec.GatewayIngressIPs...)
	t.Logf("%+v", ss1)
}

func Test5(t *testing.T) {

	type Status struct {
		finish bool
		state  *pbv3.State
	}

	status := Status{state: &pbv3.State{}}
	route := gin.Default()
	route.GET("/", func(c *gin.Context) {
		data := make(map[string]interface{})
		data["status"] = status.finish
		if status.state != nil {
			data["percent"] = status.state.Current
		}
		bs, _ := json.Marshal(data)
		fmt.Println("percentis : ", status.finish)
		fmt.Println("data is : ", string(bs))
		c.JSON(200, string(bs))
	})
	route.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, map[string]interface{}{"msg": err.Error})
			return
		}
		fmt.Println(file.Filename)
		downPath := path.Join("/tmp", file.Filename)

		var out io.Writer
		f, _ := os.OpenFile(downPath, os.O_CREATE|os.O_WRONLY, 0644)
		out = f
		defer f.Close()

		bar := pbv3.Full.Start64(c.Request.ContentLength)

		status.state.ProgressBar = bar
		// status.bar = progressbar.NewOptions(
		// 	int(c.Request.ContentLength),
		// 	progressbar.OptionSetBytes(int(c.Request.ContentLength)),
		// )
		// go func() {
		// 	for {
		// 		t.Logf("-----------------pencent: %v, bytes: %v, all: %v, finisih: %v", bar.State().CurrentPercent, bar.State().CurrentBytes, bar.State().MaxBytes, (bar.State().MaxBytes-int64(bar.State().CurrentBytes)) < 1)
		// 		time.Sleep(time.Second / 2)
		// 		status = Status{
		// 			currentPercent: bar.State().CurrentPercent,
		// 			finish:         (bar.State().MaxBytes - int64(bar.State().CurrentBytes)) < 1,
		// 		}
		// 		fmt.Printf("staus : %+v", status)
		// 	}
		// }()

		// create proxy reader
		barReader := bar.NewProxyReader(c.Request.Body)
		// Write the body to file
		_, _ = io.Copy(out, barReader)

		c.String(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
	})
	_ = route.Run()
}

func Test7(t *testing.T) {
	resp, err := http.Get("https://github.com/schollz/croc/releases/download/v4.1.4/croc_v4.1.4_Windows-64bit_GUI.zip")
	if err != nil { // TODO fanyangyang if can't create connection, download manual and upload it
		t.Fatal("get error : ", err.Error())
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create("/tmp/rainbond.tar") // TODO fanyangyang file path and generate test case
	if err != nil {
		t.Fatal("create errror: ", err.Error())
	}
	defer out.Close()
	// start new bar
	listener := OssProgressListener{TotalRwBytes: resp.ContentLength}
	fmt.Println("total is : ", resp.ContentLength)
	reader := oss.TeeReader(resp.Body, out, resp.ContentLength, &listener, nil)
	defer func() { _ = reader.Close() }()
	go func() {
		for {
			fmt.Printf("percent:%v, current: %v, total: %v \n", listener.Percent, listener.CurrentBytes, listener.TotalRwBytes)
			time.Sleep(2 * time.Second)
		}
	}()
	_, _ = io.Copy(out, reader)

	if err := os.Remove("/tmp/rainbond.tar"); err != nil {
		t.Fatal(err)
	}

}

// OssProgressListener is the progress listener
type OssProgressListener struct {
	TotalRwBytes int64
	CurrentBytes int64
	Percent      int
	Finished     bool
}

// ProgressChanged handles progress event
func (listener *OssProgressListener) ProgressChanged(event *oss.ProgressEvent) {
	switch event.EventType {
	case oss.TransferStartedEvent:
		fmt.Printf("Transfer Started.\n")
	case oss.TransferDataEvent:
		listener.CurrentBytes = event.ConsumedBytes
		if listener.TotalRwBytes != 0 {
			listener.Percent = int(100 * listener.CurrentBytes / listener.TotalRwBytes)
		}
		fmt.Printf("Transfer Data, This time consumedBytes: %d \n", event.ConsumedBytes)
	case oss.TransferCompletedEvent:
		listener.Finished = true
		fmt.Printf("Transfer Completed, This time consumedBytes: %d.\n", event.ConsumedBytes)
	case oss.TransferFailedEvent:
		fmt.Printf("Transfer Failed, This time consumedBytes: %d.\n", event.ConsumedBytes)
	default:
	}
}

func TestStat(t *testing.T) {
	now := time.Now()
	if _, err := os.Stat("/opt/rainbond/pkg/tgz/rainbond-pkg-V5.2-dev.tgz"); os.IsNotExist(err) {
		t.Log("file is not exists")
	}
	t.Log(time.Since(now))
}
