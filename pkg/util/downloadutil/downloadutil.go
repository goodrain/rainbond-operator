package downloadutil

import (
	"io"
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
}

var tmpPath = "/opt/rainbond/pkg/rainbond-pkg.tar"

// Download download
func (listener *DownloadWithProgress) Download() error {
	// Get the data
	if listener.URL == "" {
		// listener.URL = "127.0.0.1"
	}
	resp, err := http.Get(listener.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(tmpPath)
	defer out.Close()
	if err != nil {
		return err
	}
	listener.TotalRwBytes = resp.ContentLength
	logrus.Debugf("package size total is : %d", resp.ContentLength/1024/1024)

	reader := oss.TeeReader(resp.Body, nil, listener.TotalRwBytes, listener, nil)
	defer reader.Close()
	if _, err = io.Copy(out, reader); err != nil {
		return err
	}
	logrus.Debug("download finished, move file to ", listener.SavedPath)
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
