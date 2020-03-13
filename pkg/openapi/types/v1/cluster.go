package v1

import "strings"

// ComponentStatus component status
type ComponentStatus string

const (
	//ComponentStatusRunning running
	ComponentStatusRunning = "Running"
	// ComponentStatusIniting initing
	ComponentStatusIniting = "Initing"
	//ComponentStatusCreating creating
	ComponentStatusCreating = "Creating"
	// ComponentStatusTerminating terminal
	ComponentStatusTerminating = "Terminating" // TODO fanyangyang have not found this case
	// ComponentStatusFailed failed
	ComponentStatusFailed = "Failed"
)

// RbdComponentStatus rainbond component status
type RbdComponentStatus struct {
	Name string `json:"name"`

	// Total number of non-terminated pods targeted by this deployment (their labels match the selector).
	// +optional
	Replicas int32 `json:"replicas"`

	// Total number of ready pods targeted by this deployment.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas"`

	Status          ComponentStatus `json:"status"` //translate pod status to component status
	Message         string          `json:"message"`
	Reason          string          `json:"reason"`
	ISInitComponent bool            `json:"isInitComponent"`

	PodStatuses []PodStatus `json:"podStatus"`
}

//RbdComponentStatusList list of rbdComponentStatus implement sort
type RbdComponentStatusList []*RbdComponentStatus

// Len len of rbdComponentStatusList
func (l RbdComponentStatusList) Len() int {
	return len(l)
}

// Swap swap list i and j
func (l RbdComponentStatusList) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// Less list i is less list j or not
func (l RbdComponentStatusList) Less(i, j int) bool {
	return strings.Compare(l[i].Name, l[j].Name) == -1
}

// PodStatus represents information about the status of a pod, which belongs to RbdComponent.
type PodStatus struct {
	Name              string               `json:"name"`
	Phase             string               `json:"phase"`
	HostIP            string               `json:"hostIP"`
	Reason            string               `json:"reason"`
	Message           string               `json:"message"`
	ContainerStatuses []PodContainerStatus `json:"container_statuses"`
}

// PodContainerStatus -
type PodContainerStatus struct {
	ContainerID string `json:"containerID"`
	Image       string `json:"image"`
	// Specifies whether the container has passed its readiness probe.
	Ready   bool   `json:"ready"`
	State   string `json:"state"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// K8sNode holds the information about a kubernetes node.
type K8sNode struct {
	Name       string `json:"name"`
	InternalIP string `json:"internalIP" binding:"required,ipv4"`
	ExternalIP string `json:"externalIP"`
}

// AvailableNodes contains nodes available for special rainbond components to run,
// such as rbd-gateway, rbd-chaos.
type AvailableNodes struct {
	// The nodes with user-specified labels.
	SpecifiedNodes []*K8sNode `json:"specifiedNodes,omitempty"`
	// A list of kubernetes master nodes.
	MasterNodes []*K8sNode `json:"masterNodes,omitempty"`
}

// StorageClass is a List of StorageCass available in the cluster.
// StorageClass storage class
type StorageClass struct {
	Name        string `json:"name"`
	Provisioner string `json:"provisioner"`
	AccessMode  string `json:"accessMode"`
}

// ClusterStatusInfo holds the information of rainbondcluster status.
type ClusterStatusInfo struct {
	// holds some recommend nodes available for rbd-gateway to run.
	GatewayAvailableNodes *AvailableNodes `json:"gatewayAvailableNodes"`
	// holds some recommend nodes available for rbd-chaos to run.
	ChaosAvailableNodes *AvailableNodes `json:"chaosAvailableNodes"`
	StorageClasses      []*StorageClass `json:"storageClasses"`
}

// ImageHub image hub
type ImageHub struct {
	Domain    string `json:"domain"`
	Namespace string `json:"namespace"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

// Database defines the connection information of database.
type Database struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// EtcdConfig defines the configuration of etcd client.
type EtcdConfig struct {
	// Endpoints is a list of URLs.
	Endpoints []string `json:"endpoints"`
	// Secret to mount to read certificate files for tls.
	CertInfo *EtcdCertInfo `json:"certInfo"`
}

// EtcdCertInfo etcd cert info
type EtcdCertInfo struct {
	CaFile   string `json:"caFile"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

// GlobalConfigs check result
type GlobalConfigs struct {
	// Enable highly available installation, otherwise use all-in-one mode
	EnableHA          bool       `json:"enableHA"`
	ImageHub          ImageHub   `json:"imageHub"`
	RegionDatabase    Database   `json:"regionDatabase"`
	UIDatabase        Database   `json:"uiDatabase"`
	EtcdConfig        EtcdConfig `json:"etcdConfig"`
	HTTPDomain        string     `json:"HTTPDomain"`
	GatewayIngressIPs []string   `json:"gatewayIngressIPs"`
	NodesForGateways  []*K8sNode `json:"nodesForGateway" binding:"required,dive,required"`
	NodesForChaos     []*K8sNode `json:"nodesForChaos" binding:"required,dive,required"`
}

// AliyunCloudDiskCSIPluginSource represents a aliyun cloud disk CSI plugin.
// More info: https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/blob/master/docs/disk.md
type AliyunCloudDiskCSIPluginSource struct {
	// The AccessKey ID provided by Alibaba Cloud for access control.
	AccessKeyID string `json:"accessKeyID"`
	// The AccessKey Secret provided by Alibaba Cloud for access control
	AccessKeySecret string `json:"accessKeySecret"`

	MaxVolumePerNode int `json:"maxVolumePerNode" binding:"lt=16"`
}

// AliyunNasCSIPluginSource represents a aliyun cloud nas CSI plugin.
// More info: https://github.com/GLYASAI/alibaba-cloud-csi-driver/blob/master/docs/nas.md
type AliyunNasCSIPluginSource struct {
	// The AccessKey ID provided by Alibaba Cloud for access control.
	AccessKeyID string `json:"accessKeyID"`
	// The AccessKey Secret provided by Alibaba Cloud for access control
	AccessKeySecret string `json:"accessKeySecret"`
}

// NFSCSIPluginSource represents a nfs CSI plugin.
// More info: https://github.com/kubernetes-incubator/external-storage/tree/master/nfs
type NFSCSIPluginSource struct {
}

// CSIPluginSource represents the source of a csi driver to create.
// Only one of its members may be specified.
type CSIPluginSource struct {
	// AliyunCloudDiskCSIPluginSource represents a aliyun cloud disk CSI plugin.
	// More info: https://github.com/kubernetes-sigs/alibaba-cloud-csi-driver/blob/master/docs/disk.md
	AliyunCloudDisk *AliyunCloudDiskCSIPluginSource `json:"aliyunCloudDisk"`
	// AliyunNasCSIPluginSource represents a aliyun cloud nas CSI plugin.
	// More info: https://github.com/GLYASAI/alibaba-cloud-csi-driver/blob/master/docs/nas.md
	AliyunNas *AliyunNasCSIPluginSource `json:"aliyunNas"`
	// NFSCSIPluginSource represents a nfs CSI plugin.
	// More info: https://github.com/kubernetes-incubator/external-storage/tree/master/nfs
	NFS *NFSCSIPluginSource `json:"nfs,omitempty"`
}

// StorageClassParameters describes the parameters for a class of storage for
// which PersistentVolumes can be dynamically provisioned.
type StorageClassParameters struct {
	// Provisioner indicates the type of the provisioner.
	Provisioner string `json:"provisioner"`
	// Parameters holds the parameters for the provisioner that should
	// create volumes of this storage class.
	// +optional
	Parameters map[string]string `json:"parameters"`
}

// RainbondVolume -
type RainbondVolume struct {
	// Specify the name of the storageClass directly
	StorageClassName string `json:"storageClassName"`
	// Specify the storageClass parameter to directly create the corresponding StorageClass
	StorageClassParameters *StorageClassParameters `json:"storageClassParameters"`

	CSIPlugin *CSIPluginSource `json:"csiPlugin"`
}

// RainbondVolumes contains information about rainbondvolume, including RWX and RWO.
// Used to prepare StorageClass for rainbond application.
type RainbondVolumes struct {
	RWX *RainbondVolume `json:"RWX" binding:"required"`
	RWO *RainbondVolume `json:"RWO"`
}

// ClusterInstallReq -
type ClusterInstallReq struct {
	RainbondVolumes *RainbondVolumes `json:"rainbondVolumes" binding:"required"`
}
