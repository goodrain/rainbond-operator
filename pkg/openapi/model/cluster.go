package model

// GlobalConfigs check result
type GlobalConfigs struct {
	// default goodrain.me
	ImageHub *ImageHub `json:"imageHub,omitempty"`
	// the storage class that rainbond component will be used.
	// rainbond-operator will create one if StorageClassName is empty
	StorageClassName string `json:"storageClassName,omitempty"`
	// the region database information that rainbond component will be used.
	// rainbond-operator will create one if DBInfo is empty
	RegionDatabase *Database `json:"regionDatabase,omitempty"`
	// the ui database information that rainbond component will be used.
	// rainbond-operator will create one if DBInfo is empty
	UIDatabase *Database `json:"uiDatabase,omitempty"`
	// the etcd connection information that rainbond component will be used.
	// rainbond-operator will create one if EtcdConfig is empty
	EtcdConfig *EtcdConfig `json:"etcdConfig,omitempty"`
	// KubeAPIHost must be a host string, a host:port pair, or a URL to the base of the apiserver.
	// If a URL is given then the (optional) Path of that URL represents a prefix that must
	// be appended to all request URIs used to access the apiserver. This allows a frontend
	// proxy to easily relocate all of the apiserver endpoints.
	KubeAPIHost    string            `json:"kubeAPIHost,omitempty"`
	NodeAvailPorts []*NodeAvailPorts `json:"availPorts,omitempty"`
}

// RbdComponent rbd component
type RbdComponent struct {
}

// NodeAvailPorts aval port
type NodeAvailPorts struct {
	Ports    []int  `json:"port,omitempty"`
	NodeIP   string `json:"nodeIP,omitempty"`
	NodeName string `json:"nodeName,omitempty"`
}

// ImageHub image hub
type ImageHub struct {
	Domain    string `json:"domain,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
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
	// Whether to use tls to connect to etcd
	UseTLS bool `json:"useTLS"`
	// Secret to mount to read certificate files for tls.
	CertInfo *EtcdCertInfo `json:"certInfo"`
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
