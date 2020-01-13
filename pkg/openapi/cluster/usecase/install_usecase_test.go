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
	"sync"
	"testing"
	"time"

	v1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/cheggaaa/pb"
	pbv3 "github.com/cheggaaa/pb/v3"
	"github.com/gin-gonic/gin"
	"github.com/schollz/progressbar/v2"
)

func Test_downloadFile(t *testing.T) {
	if _, err := os.Stat("/tmp/rainbond.tar"); os.IsNotExist(err) {
		t.Log("do not exists, downloading...")
		ic := InstallUseCaseImpl{}
		if err := ic.downloadFile("/tmp/rainbond.tar", "http://192.168.200.2"); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Log("already exists, do not download again")
	}
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
	io.Copy(writer, barReader)
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
	_, err = io.Copy(out, barReader)

	// t.Log("", bar.State().IsFinished())

	if err := os.Remove("/tmp/rainbond.tar"); err != nil {
		t.Fatal(err)
	}
}

func Test3(t *testing.T) {
	urlToGet := "https://github.com/schollz/croc/releases/download/v4.1.4/croc_v4.1.4_Windows-64bit_GUI.zip"
	req, _ := http.NewRequest("GET", urlToGet, nil)
	resp, _ := http.DefaultClient.Do(req)
	defer resp.Body.Close()

	var out io.Writer
	f, _ := os.OpenFile("croc_v4.1.4_Windows-64bit_GUI.zip", os.O_CREATE|os.O_WRONLY, 0644)
	out = f
	defer f.Close()

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
	out = io.MultiWriter(out, bar)

	// io.Copy(out, resp.Body)
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
		currentPercent float64
		finish         bool
		state          *pbv3.State
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
		_, err = io.Copy(out, barReader)

		c.String(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
	})
	route.Run()
}

func Test6(t *testing.T) {
	state := &pbv3.State{}
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
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
		state.ProgressBar = bar
		// create proxy reader
		barReader := bar.NewProxyReader(resp.Body)
		// Write the body to file
		_, err = io.Copy(out, barReader)

		// t.Log("", bar.State().IsFinished())

		if err := os.Remove("/tmp/rainbond.tar"); err != nil {
			t.Fatal(err)
		}
		wg.Done()
	}()

	go func() {
		for {
			fmt.Println("running...")
			if state.ProgressBar != nil {
				fmt.Println("current is : ", state.Current())
				if state.IsFinished() {
					wg.Done()
				}
			}
			time.Sleep(2 * time.Second)
		}
	}()

	wg.Wait()
}
