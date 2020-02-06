package model

// GlobalConfigs check result
type GlobalConfigs struct {
	ImageHub          ImageHub      `json:"imageHub"`
	RegionDatabase    Database      `json:"regionDatabase"`
	UIDatabase        Database      `json:"uiDatabase"`
	EtcdConfig        EtcdConfig    `json:"etcdConfig"`
	GatewayNodes      []GatewayNode `json:"gatewayNodes"`
	HTTPDomain        HTTPDomain    `json:"HTTPDomain"`
	GatewayIngressIPs []string      `json:"gatewayIngressIPs"`
	Storage           Storage       `json:"storage"`
}

// RbdComponent rbd component
type RbdComponent struct {
}

// HTTPDomain http domain
type HTTPDomain struct {
	Default bool   `json:"default"`
	Custom  string `json:"custom"`
}

// GatewayNode gateway
type GatewayNode struct {
	NodeName string `json:"nodeName"`
	NodeIP   string `json:"nodeIP"`
	Ports    []int  `json:"ports"`
	Selected bool   `json:"selected"`
}

// Storage storage
type Storage struct {
	Default          bool          `json:"default"`
	StorageClassName string        `json:"storageClassName"`
	Opts             []StorageOpts `json:"opts"`
}

// StorageOpts storage opts
type StorageOpts struct {
	Name        string `json:"name"`
	Provisioner string `json:"provisioner"`
}

// NodeAvailPorts aval port
type NodeAvailPorts struct {
	Ports    []int  `json:"port"`
	NodeIP   string `json:"nodeIP"`
	NodeName string `json:"nodeName"`
}

// ImageHub image hub
type ImageHub struct {
	Default   bool   `json:"default" validate:"default|required"`
	Domain    string `json:"domain"`
	Namespace string `json:"namespace"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

// Database defines the connection information of database.
type Database struct {
	Default  bool   `json:"default" validate:"default|required"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// EtcdConfig defines the configuration of etcd client.
type EtcdConfig struct {
	Default bool `json:"default" validate:"default|required"`
	// Endpoints is a list of URLs.
	Endpoints []string `json:"endpoints"`
	// Whether to use tls to connect to etcd
	UseTLS bool `json:"useTLS"`
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
