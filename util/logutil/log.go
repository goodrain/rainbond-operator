package logutil

import (
	"bytes"
	"context"
	"encoding/json"
	dclient "github.com/docker/docker/client"
	"net/http"
	"strings"
	"time"
)

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

type DockerInfo struct {
	ClientVersion string        `json:"client_version"`
	Server        *DockerServer `json:"server"`
}

type ClusterInfo struct {
	Status string `json:"status"`
}

type RegionInfo struct {
	Status string `json:"status"`
}

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

func GetDockerInfo() (info *DockerInfo, err error) {
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

func SendLog(req *LogCollectRequest) {
	data, err := json.Marshal(req)
	if err != nil {
		return
	}

	client := http.Client{
		Timeout: 1 * time.Second,
	}

	resp, err := client.Post("https://log.rainbond.com/logCollect", "application/json",
		bytes.NewBuffer(data))
	if resp != nil {
		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)
	}
}
