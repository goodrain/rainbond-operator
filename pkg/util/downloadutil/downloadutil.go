package downloadutil

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"io"
	"net/http"
	"os"
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
	resp, err := http.Get(listener.URL)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	// Create the file
	out, err := os.Create(listener.SavedPath)
	defer out.Close()
	if err != nil {
		return err
	}
	listener.TotalRwBytes = resp.ContentLength

	reader := oss.TeeReader(resp.Body, out, listener.TotalRwBytes, listener, nil)
	defer reader.Close()
	_, err = io.Copy(out, reader)
	return err
}

// ProgressChanged handles progress event
func (listener *DownloadWithProgress) ProgressChanged(event *oss.ProgressEvent) {
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
