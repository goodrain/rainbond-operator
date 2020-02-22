package constants

const (
	//Namespace namespace
	Namespace = "rbd-system"
	// DefImageRepositoryDomain is the default domain name of the mirror repository that Rainbond is installed.
	DefImageRepositoryDomain = "goodrain.me"
	//DefInstallPkgDestPath  Default destination path of the installation package extraction.
	DefInstallPkgDestPath = "/tmp/DefInstallPkgDestPath"
	//RainbondClusterName rainbond cluster resource name
	RainbondClusterName = "rainbondcluster"
	//RainbondPackageName rainbond package resource name
	RainbondPackageName = "rainbondpackage"
	//DefStorageClass -
	DefStorageClass = "rbd-nfs"
	//DefImageRepository -
	DefImageRepository = "goodrain.me"
	//GrDataPVC -
	GrDataPVC = "rbd-component-grdata"
	// CachePVC
	CachePVC = "rbd-chaos-cache"
	// SpecialGatewayLabelKey is a special node label, used to specify where to install the rbd-gateway
	SpecialGatewayLabelKey = "rainbond.io/gateway"
	// SpecialChaosLabelKey is a special node label, used to specify where to install the rbd-chaos
	SpecialChaosLabelKey = "rainbond.io/chaos"
)
