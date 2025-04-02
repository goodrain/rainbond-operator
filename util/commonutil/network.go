package commonutil

import (
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

// GetLatency Get URL latency
func GetLatency(url string) (bool, time.Duration) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", url+":443", 2*time.Second)
	if err != nil {
		logrus.Debugf("test registry %s error: %v", url, err)
		return false, 0
	}
	defer conn.Close()
	return true, time.Since(start)
}

type Result struct {
	URL     string
	Success bool
	Latency time.Duration
}

// CompareLatency Compare the latency of multiple urls and return the address with the least latency
func CompareLatency(urls ...string) string {
	ch := make(chan Result, len(urls))
	// concurrent execution Latency Test
	for _, url := range urls {
		go func(u string) {
			success, latency := GetLatency(u)
			ch <- Result{u, success, latency}
		}(url)
	}

	var validResults []Result
	timeout := time.After(3 * time.Second)
	for i := 0; i < len(urls); i++ {
		select {
		case res := <-ch:
			if res.Success {
				validResults = append(validResults, res)
				logrus.Debugf("successful detection %s latency: %v", res.URL, res.Latency)
			}
		case <-timeout:
			logrus.Warn("partial address latency detection timeout")
			break
		}
	}

	// 决策逻辑
	if len(validResults) == 0 {
		logrus.Warn("all addresses are unavailable")
		return ""
	}

	// 选择延迟最小的可用仓库
	best := validResults[0]
	for _, res := range validResults[1:] {
		if res.Latency < best.Latency {
			best = res
		}
	}
	logrus.Infof("optimal address: %s (latency: %v)", best.URL, best.Latency)
	return best.URL
}
