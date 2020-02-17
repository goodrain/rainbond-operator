package model

// GlobalConfigs check result
type GlobalConfigs struct {
	ImageHub             ImageHub             `json:"imageHub"`
	RegionDatabase       Database             `json:"regionDatabase"`
	UIDatabase           Database             `json:"uiDatabase"`
	EtcdConfig           EtcdConfig           `json:"etcdConfig"`
	GatewayNodes         []GatewayNode        `json:"gatewayNodes"`
	HTTPDomain           string               `json:"HTTPDomain"`
	GatewayIngressIPs    []string             `json:"gatewayIngressIPs"`
	Storage              Storage              `json:"storage"`
	RainbondShareStorage RainbondShareStorage `json:"rainbondShareStorage"`
}

// RbdComponent rbd component
type RbdComponent struct {
}

// GatewayNode gateway
type GatewayNode struct {
	NodeName string `json:"nodeName"`
	NodeIP   string `json:"nodeIP"`
	Ports    []int  `json:"ports"`
}

// Storage storage
type Storage struct {
	Name        string `json:"name"`
	Provisioner string `json:"provisioner"`
}

// StorageOpts storage opts
type StorageOpts struct {
	Name        string `json:"name"`
	Provisioner string `json:"provisioner"`
}

// FstabLine represents a line in file /etc/fstab.
type FstabLine struct {
	Device     string `json:"fileSystem,omitempty"`
	MountPoint string `json:"mountPoint,omitempty"`
	Type       string `json:"type,omitempty"`
	Options    string `json:"options,omitempty"`
	Dump       int    `json:"dump,omitempty"`
	Pass       int    `json:"pass,omitempty"`
}

// RainbondShareStorage -
type RainbondShareStorage struct {
	StorageClassName string     `json:"storageClassName"`
	FstabLine        *FstabLine `json:"fstabLine"`
}

// NodeAvailPorts aval port
type NodeAvailPorts struct {
	Ports    []int  `json:"port"`
	NodeIP   string `json:"nodeIP"`
	NodeName string `json:"nodeName"`
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
	CertInfo EtcdCertInfo `json:"certInfo"`
}

// EtcdCertInfo etcd cert info
type EtcdCertInfo struct {
	CaFile   string `json:"caFile"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
}

// ComponseInfo componse info
type ComponseInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	HealthCount int    `json:"healthCount"`
	TotalCount  int    `json:"totalCount"`
	Message     string `json:"message"`
}

// InstallStatus install status
type InstallStatus struct {
	StepName string `json:"stepName"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Message  string `json:"message"`
	Reason   string `json:"reason"`
}

// StatusRes StatusRes
type StatusRes struct {
	FinalStatus string          `json:"finalStatus"`
	StatusList  []InstallStatus `json:"statusList"`
}

// ClusterStatus cluster global status
type ClusterStatus struct {
	FinalStatus GlobalStatus `json:"final_status"`
	ClusterInfo ClusterInfo  `json:"clusterInfo"`
}

// ClusterInfo cluster info used for config
type ClusterInfo struct {
	NodeAvailPorts []NodeAvailPorts `json:"nodeAvailPorts"`
	Storage        []Storage        `json:"storage"`
	EnterpriseID   string           `json:"enterpriseID"` // enterprise's uuid
	InstallID      string           `json:"installID"`    // install uuid
	InstallVersion string           `json:"installVersion"`
}
