package controller

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster/mock"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"github.com/goodrain/rainbond-operator/pkg/util/ginutil"
	"github.com/pquerna/ffjson/ffjson"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestClusterInfo404(t *testing.T) {
	ctrl := gomock.NewController(t)

	ucase := mock.NewMockUsecase(ctrl)
	ucase.EXPECT().StatusInfo().Return(nil, bcode.NotFound)

	clusterUcase := mock.NewMockIClusterUcase(ctrl)
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

	clusterUcase := mock.NewMockIClusterUcase(ctrl)
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
		GatewayAvailableNodes: &v1.AvailableNodes{
			SpecifiedNodes: []*v1.K8sNode{
				{
					Name:       "foo",
					InternalIP: "1.1.1.1",
				},
			},
		},
		ChaosAvailableNodes: &v1.AvailableNodes{
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

	clusterUcase := mock.NewMockIClusterUcase(ctrl)
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

func TestClusterInfoRequest(t *testing.T) {
	tests := []struct {
		name string
		data *v1.GlobalConfigs
		want bcode.Coder
	}{
		{
			name: "no nodes for chaos",
			data: &v1.GlobalConfigs{
				NodesForGateways: []*v1.K8sNode{
					{
						Name: "foo",
					},
				},
			},
			want: bcode.BadRequest,
		},
		{
			name: "missing internal ip",
			data: &v1.GlobalConfigs{
				NodesForGateways: []*v1.K8sNode{
					{
						Name: "foo",
					},
				},
				NodesForChaos: []*v1.K8sNode{
					{
						Name:       "bar",
						InternalIP: "1.1.1.1",
					},
				},
			},
			want: bcode.BadRequest,
		},
		{
			name: "wrong internal ip",
			data: &v1.GlobalConfigs{
				NodesForGateways: []*v1.K8sNode{
					{
						Name: "foo",
					},
				},
				NodesForChaos: []*v1.K8sNode{
					{
						Name:       "bar",
						InternalIP: "1.1.1.1",
					},
				},
			},
			want: bcode.BadRequest,
		},
	}

	path := "/cluster/configs"
	for idx := range tests {
		tc := tests[idx]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			usecase := mock.NewMockUsecase(ctrl)
			usecase.EXPECT().CompleteNodes(tc.data.NodesForGateways, true).Return(nil, nil)
			usecase.EXPECT().CompleteNodes(tc.data.NodesForChaos, false).Return(nil, nil)

			installUcase := mock.NewMockInstallUseCase(ctrl)
			statusRes := model.StatusRes{}
			installUcase.EXPECT().InstallStatus().Return(statusRes, nil)

			configUcase := mock.NewMockGlobalConfigUseCase(ctrl)
			configUcase.EXPECT().UpdateGlobalConfig(gomock.Any()).Return(nil)

			clusterUcase := mock.NewMockIClusterUcase(ctrl)
			clusterUcase.EXPECT().Install().Return(installUcase)
			clusterUcase.EXPECT().GlobalConfigs().Return(configUcase)
			clusterUcase.EXPECT().Cluster().AnyTimes().Return(usecase)

			cc := &ClusterController{
				clusterUcase: clusterUcase,
			}

			r := gin.Default()
			// setup router
			r.PUT(path, cc.UpdateConfig)

			body, _ := ffjson.Marshal(tc.data)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("PUT", path, bytes.NewBuffer(body))
			r.ServeHTTP(w, req)

			assert.Equal(t, tc.want.Code(), w.Code)
		})
	}
}
