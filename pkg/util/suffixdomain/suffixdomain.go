package suffixdomain

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/util/uuid"
)

var (
	id        string
	secretKey string
)

func init() {
	id = initFileValue("/opt/rainbond/.init/uuid")
	secretKey = initFileValue("/opt/rainbond/.init/secretkey")
}

func initFileValue(filePath string) string {
	value := ""
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// file does not exists
		value = string(uuid.NewUUID())
		ioutil.WriteFile(filePath, []byte(value), 0644)
	} else {
		info, err := ioutil.ReadFile(filePath)
		if err != nil {
			value = string(uuid.NewUUID())
		} else {
			value = string(info)
		}
		if strings.TrimSpace(value) == "" {
			value = string(uuid.NewUUID())
			ioutil.WriteFile(filePath, []byte(value), 0644)
		}
	}
	return value
}

// GenerateDomain generate suffix domain
func GenerateDomain(iip string) (string, error) {
	body := make(url.Values)
	body["ip"] = []string{iip}
	body["uuid"] = []string{id}
	body["type"] = []string{"False"}
	body["auth"] = []string{secretKey}

	resp, err := http.PostForm("http://domain.grapps.cn/domain/new", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	fmt.Println("domain is : ", string(data))

	return string(data), nil
}
