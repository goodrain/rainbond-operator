package controller

import (
	"errors"
	"github.com/golang/mock/gomock"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster/mock"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"github.com/goodrain/rainbond-operator/pkg/util/ginutil"
	"github.com/pquerna/ffjson/ffjson"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestClusterInfo404(t *testing.T) {
	ctrl := gomock.NewController(t)

	ucase := mock.NewMockUsecase(ctrl)
	ucase.EXPECT().StatusInfo().Return(nil, bcode.NotFound)

	clusterUcase := mock.NewMockIClusterCase(ctrl)
	clusterUcase.EXPECT().Cluster().Return(ucase)

	cc := &ClusterController{
		clusterUcase: clusterUcase,
	}

	r := gin.Default()
	// setup router
	r.GET("/cluster/status-info", cc.ClusterStatusInfo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cluster/status-info", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 404, w.Code)
}

func TestClusterInfo500(t *testing.T) {
	ctrl := gomock.NewController(t)

	ucase := mock.NewMockUsecase(ctrl)
	ucase.EXPECT().StatusInfo().Return(nil, errors.New("foobar"))

	clusterUcase := mock.NewMockIClusterCase(ctrl)
	clusterUcase.EXPECT().Cluster().Return(ucase)

	cc := &ClusterController{
		clusterUcase: clusterUcase,
	}

	r := gin.Default()
	// setup router
	r.GET("/cluster/status-info", cc.ClusterStatusInfo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cluster/status-info", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)
}

func TestClusterInfoOK(t *testing.T) {
	statusInfo := &v1.ClusterStatusInfo{
		GatewayAvailableNodes: v1.AvailableNodes{
			SpecifiedNodes: []*v1.K8sNode{
				{
					Name:       "foo",
					InternalIP: "1.1.1.1",
				},
			},
		},
		ChaosAvailableNodes: v1.AvailableNodes{
			MasterNodes: []*v1.K8sNode{
				{
					Name:       "foo",
					InternalIP: "1.1.1.1",
				},
			},
		},
	}

	ctrl := gomock.NewController(t)

	ucase := mock.NewMockUsecase(ctrl)
	ucase.EXPECT().StatusInfo().Return(statusInfo, nil)

	clusterUcase := mock.NewMockIClusterCase(ctrl)
	clusterUcase.EXPECT().Cluster().Return(ucase)

	cc := &ClusterController{
		clusterUcase: clusterUcase,
	}

	r := gin.Default()
	// setup router
	r.GET("/cluster/status-info", cc.ClusterStatusInfo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/cluster/status-info", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	var result ginutil.Result
	err := ffjson.Unmarshal(w.Body.Bytes(), &result)
	assert.Nil(t, err)

	bytes, err := ffjson.Marshal(result.Data)
	assert.Nil(t, err)
	var info v1.ClusterStatusInfo
	err = ffjson.Unmarshal(bytes, &info)
	assert.Nil(t, err)

	assert.EqualValues(t, statusInfo, &info)
}
