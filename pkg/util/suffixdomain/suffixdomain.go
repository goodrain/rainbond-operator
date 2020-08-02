package suffixdomain

import (
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
)

var defaultdomain = "offline.appdomain"

// GenerateDomain generate suffix domain
func GenerateDomain(iip, id, secretKey string) (string, error) {
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
	if resp.StatusCode == 200 {
		return string(data), nil
	}
	logrus.Errorf("cenerate domain failure %s", string(data))
	return defaultdomain, nil
}
