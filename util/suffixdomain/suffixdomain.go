package suffixdomain

import (
	"io/ioutil"
	"net/http"
	"net/url"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

// GenerateDomain generate suffix domain
func GenerateDomain(iip, id, secretKey string) (string, error) {
	body := make(url.Values)
	body["ip"] = []string{iip}
	body["uuid"] = []string{id}
	body["type"] = []string{"False"}
	body["auth"] = []string{secretKey}

	resp, err := http.PostForm("http://domain.wtapps.cn/domain/new", body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GenerateSuffixConfigMap -
func GenerateSuffixConfigMap(name, namespace string) *v1.ConfigMap {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"uuid": string(uuid.NewUUID()),
			"auth": string(uuid.NewUUID()),
		},
	}
	return cm
}
