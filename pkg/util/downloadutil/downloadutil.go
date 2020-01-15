package downloadutil

import (
	"io"
	"net/http"
	"os"

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
	out, err := os.Create(listener.SavedPath)
	defer out.Close()
	if err != nil {
		return err
	}
	listener.TotalRwBytes = resp.ContentLength

	reader := oss.TeeReader(resp.Body, nil, listener.TotalRwBytes, listener, nil)
	defer reader.Close()
	_, err = io.Copy(out, reader)
	return err
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
		logrus.Debugf("Transfer Data, This time consumedBytes: %d \n", event.ConsumedBytes)
	case oss.TransferCompletedEvent:
		listener.Finished = true
		logrus.Debugf("Transfer Completed, This time consumedBytes: %d.\n", event.ConsumedBytes)
	case oss.TransferFailedEvent:
		logrus.Debugf("Transfer Failed, This time consumedBytes: %d.\n", event.ConsumedBytes)
	default:
	}
}
