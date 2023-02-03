package logutil

import (
	"context"
	"runtime"
	"strings"
	"time"

	dclient "github.com/docker/docker/client"
	"github.com/wutong-paas/wutong-operator/util/containerutil"
)

// LogCollectRequest define info which should be collect
type LogCollectRequest struct {
	EID           string       `json:"eid"`
	Version       string       `json:"version"`
	OS            string       `json:"os"`
	OSArch        string       `json:"os_arch"`
	KernelVersion string       `json:"kernel_version"`
	DockerInfo    *DockerInfo  `json:"docker_info"`
	ClusterInfo   *ClusterInfo `json:"cluster_info"`
	RegionInfo    *RegionInfo  `json:"region_info"`
}

// DockerInfo define infos about docker which should be collect
type DockerInfo struct {
	ClientVersion string        `json:"client_version"`
	Server        *DockerServer `json:"server"`
}

// ClusterInfo define infos about kubernetes cluster which should be collect
type ClusterInfo struct {
	Status string `json:"status"`
}

// RegionInfo define infos about wutong region which should be collect
type RegionInfo struct {
	Status string `json:"status"`
	Pods   []*Pod `json:"pods"`
}

// PodEvent define infos about pod`s event which should be collect
type PodEvent struct {
	Type    string `json:"type"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
	From    string `json:"from"`
	Age     string `json:"age"`
}

// Pod define infos about pod which should be collect
type Pod struct {
	Name   string      `json:"name"`
	Status string      `json:"status"`
	Ready  string      `json:"ready"`
	Log    string      `json:"log"`
	Events []*PodEvent `json:"events"`
}

// DockerServer define infos about DockerServer which should be collect
type DockerServer struct {
	Version         string `json:"version"`
	APIVersion      string `json:"api_version"`
	OS              string `json:"os"`
	OSArch          string `json:"os_arch"`
	Driver          string `json:"driver"`
	IPv4Forwarding  bool   `json:"ipv4_forwarding"`
	CgroupDriver    string `json:"cgroup_driver"`
	CgroupVersion   string `json:"cgroup_version"`
	KernelVersion   string `json:"kernel_version"`
	OperatingSystem string `json:"operating_system"`
	NCPU            int    `json:"ncpu"`
	MemTotal        int64  `json:"mem_total"`
	DockerRootDir   string `json:"docker_root_dir"`
	Name            string `json:"name"`
}

// GetDockerInfo -
func GetDockerInfo() (info *DockerInfo, err error) {
	if containerutil.GetContainerRuntime() != containerutil.ContainerRuntimeDocker {
		server := &DockerServer{
			OS:     runtime.GOOS,
			OSArch: runtime.GOARCH,
			NCPU:   runtime.NumCPU(),
		}
		return &DockerInfo{
			ClientVersion: containerutil.ContainerRuntimeContainerd,
			Server:        server,
		}, nil
	}
	cli, err := dclient.NewClientWithOpts(dclient.FromEnv)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	sv, err := cli.ServerVersion(ctx)
	if err != nil {
		var apiVersion string
		if strings.Contains(err.Error(), "Maximum supported API version is") {
			msg := strings.Split(err.Error(), " ")
			if len(msg) > 0 {
				apiVersion = msg[len(msg)-1]
				cli, err = dclient.NewClientWithOpts(dclient.WithVersion(apiVersion))
				if err != nil {
					return nil, err
				}
			}
		}
	}
	dInfo, err := cli.Info(ctx)
	if err != nil {
		return nil, err
	}
	server := &DockerServer{
		Version:         sv.Version,
		APIVersion:      sv.APIVersion,
		OS:              sv.Os,
		OSArch:          sv.Arch,
		Driver:          dInfo.Driver,
		IPv4Forwarding:  dInfo.IPv4Forwarding,
		CgroupDriver:    dInfo.CgroupDriver,
		CgroupVersion:   dInfo.CgroupVersion,
		KernelVersion:   dInfo.KernelVersion,
		OperatingSystem: dInfo.OperatingSystem,
		NCPU:            dInfo.NCPU,
		MemTotal:        dInfo.MemTotal / 1024 / 1024,
		DockerRootDir:   dInfo.DockerRootDir,
		Name:            dInfo.Name,
	}
	info = &DockerInfo{
		Server:        server,
		ClientVersion: cli.ClientVersion(),
	}
	return info, nil
}

// SendLog function for send log to request-server
func SendLog(req *LogCollectRequest) {
	// data, err := json.Marshal(req)
	// if err != nil {
	// 	return
	// }
	// client := http.Client{
	// 	Timeout: 1 * time.Second,
	// }
	// todo
	// resp, _ := client.Post("https://log.wutong.example.com/logCollect", "application/json",
	// 	bytes.NewBuffer(data))
	// if resp != nil {
	// 	var res map[string]interface{}
	// 	json.NewDecoder(resp.Body).Decode(&res)
	// }
}
