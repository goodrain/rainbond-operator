package downloadutil

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/sirupsen/logrus"
)

// DownloadWithProgress is the progress listener
type DownloadWithProgress struct {
	TotalRwBytes int64
	CurrentBytes int64
	Percent      int
	Finished     bool
	URL          string
	SavedPath    string
	Wanted       string
}

var tmpPath = "/opt/rainbond/pkg/rainbond-pkg.tar"

// Download download
func (listener *DownloadWithProgress) Download() error {
	// Get the data
	resp, err := http.Get(listener.URL)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Create the file
	out, err := os.Create(tmpPath)
	defer func() { _ = out.Close() }()
	if err != nil {
		return err
	}
	listener.TotalRwBytes = resp.ContentLength
	logrus.Debugf("package size total is : %d", resp.ContentLength/1024/1024)

	reader := oss.TeeReader(resp.Body, nil, listener.TotalRwBytes, listener, nil)
	defer func() { _ = reader.Close() }()
	if _, err = io.Copy(out, reader); err != nil {
		return err
	}
	logrus.Debug("download finished, check md5")
	if err := listener.checkMD5(out); err != nil {
		return err
	}
	logrus.Debug("check md5 finished, move file to ", listener.SavedPath)
	dir := path.Dir(listener.SavedPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	if err = os.Rename(tmpPath, listener.SavedPath); err != nil {
		return err
	}
	return nil
}

// ProgressChanged handles progress event
func (listener *DownloadWithProgress) ProgressChanged(event *oss.ProgressEvent) {
	switch event.EventType {
	case oss.TransferStartedEvent:
		logrus.Debug("Transfer Started.\n")
	case oss.TransferDataEvent:
		listener.CurrentBytes = event.ConsumedBytes
		if listener.TotalRwBytes != 0 {
			listener.Percent = int(100 * listener.CurrentBytes / listener.TotalRwBytes)
		}
		logrus.Debugf("Transfer Data,TotalBytes: %d This time consumedBytes: %d \n", listener.TotalRwBytes, event.ConsumedBytes)
	case oss.TransferCompletedEvent:
		listener.Finished = true
		logrus.Debugf("Transfer Completed, This time consumedBytes: %d.\n", event.ConsumedBytes)
	case oss.TransferFailedEvent:
		logrus.Debugf("Transfer Failed, This time consumedBytes: %d.\n", event.ConsumedBytes)
	default:
	}
}

func (listener *DownloadWithProgress) checkMD5(target *os.File) error {
	md5hash := md5.New()
	if _, err := io.Copy(md5hash, target); err != nil {
		fmt.Println("Copy", err)
		return fmt.Errorf("prepare down file md5 error: %s", err.Error())
	}
	MD5Str := hex.EncodeToString(md5hash.Sum(nil))
	wanted := listener.GetWanted()
	if MD5Str != wanted {
		return fmt.Errorf("download file md5: %s is not equal to wanted : %s", MD5Str, wanted)
	}
	return nil
}

func (listener *DownloadWithProgress) GetWanted() string {
	return listener.Wanted
}

type md5Helper interface {
	GetWanted() string
}

// OnlineMD5 get md5 online
type OnlineMD5 struct {
	wanted string
	URL    string
}

func (h *OnlineMD5) GetWanted() string {
	resp, err := http.Get(h.URL)
	if err != nil {
		logrus.Error("get md5 error: ", err.Error())
		return ""
	}
	defer resp.Body.Close()
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Error("read md5 error: ", err.Error())
		return ""
	}
	h.wanted = hex.EncodeToString(bs)
	return h.wanted
}
