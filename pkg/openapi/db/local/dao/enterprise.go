package dao

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/goodrain/rainbond-operator/pkg/util/uuidutil"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// EnterpriseDaoImpl enterprise dao impl
type EnterpriseDaoImpl struct {
	InitPath string
}

var enterpriseLog = logf.Log.WithName("enterprise dao")

// EnterpriseID enterprise id
func (impl *EnterpriseDaoImpl) EnterpriseID() (string, error) {
	enterprise := path.Join(impl.InitPath, "enterprise")
	bs, err := ioutil.ReadFile(enterprise)

	if err != nil {
		enterpriseLog.Error(err, fmt.Sprintf("read enterprise id from file: %s failed: %s, ignore it, rewrite it", enterprise, err.Error()))
		enterpriseID := uuidutil.NewUUID()
		enterpriseIDBytes, err := hex.DecodeString(enterpriseID)
		if err != nil {
			enterpriseLog.Error(err, "decode enterprise id[%s] failed: %s", enterpriseID, err.Error())
			return "", err
		}
		if err := ioutil.WriteFile(enterprise, enterpriseIDBytes, os.ModePerm); err != nil {
			enterpriseLog.Error(err, "write enterprise id[%s] failed: %s", enterpriseID, err.Error())
			return "", err
		}
		return enterpriseID, nil
	}

	return hex.EncodeToString(bs), nil
}
