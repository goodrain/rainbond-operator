package commonutil

import (
	"bufio"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"strings"
)

const (
	//StartOfSection writ hosts start
	StartOfSection = "# Mirror repository and IP mapping"
	//EndOfSection writ hosts end
	EndOfSection = "# End of Section"
	eol          = "\n"
)

// WriteHosts set rainbond imagehub and ip to local host file
func WriteHosts(hostspath, ip string) {
	// open hostfile in operator
	hostFile, err := os.OpenFile(hostspath, os.O_RDWR|os.O_APPEND, 0777)
	if err != nil {
		logrus.Error("open host file error", err)
		return
	}
	defer hostFile.Close()
	//  check ip and rainbond hub url is exist
	r := bufio.NewReader(hostFile)
	for {
		line, err := r.ReadString('\n')
		line = strings.TrimSpace(line)
		if err != nil && err != io.EOF {
			logrus.Error("read line to host file error", err)
			panic(err)
		}
		if err == io.EOF {
			logrus.Error("read line to host file EOF", err)
			break
		}
		if line == StartOfSection {
			return
		}
	}
	// add rainbond hub url if not exist
	lines := []string{
		eol + StartOfSection + eol,
		ip + " " + "goodrain.me" + eol,
		ip + " " + "region.goodrain.me" + eol,
		EndOfSection + eol,
	}
	writer := bufio.NewWriter(hostFile)
	for _, line := range lines {
		if _, err := writer.WriteString(line); err != nil {
			logrus.Error("write line to host file error", err)
		}
	}
	writer.Flush()
}
