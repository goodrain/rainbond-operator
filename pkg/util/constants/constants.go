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
	//DefImageRepository -
	DefImageRepository = "goodrain.me"
	//GrDataPVC -
	GrDataPVC = "rbd-cpt-grdata"
	// GrShareDataPVC -
	GrShareDataPVC = "rbd-share-grdata"
	// CachePVC -
	CachePVC = "rbd-chaos-cache"
	// SpecialGatewayLabelKey is a special node label, used to specify where to install the rbd-gateway
	SpecialGatewayLabelKey = "rainbond.io/gateway"
	// SpecialChaosLabelKey is a special node label, used to specify where to install the rbd-chaos
	SpecialChaosLabelKey = "rainbond.io/chaos"

	// AliyunCSIDiskPlugin name for aliyun csi disk plugin
	AliyunCSIDiskPlugin = "aliyun-csi-disk-plugin"
	// AliyunCSIDiskProvisioner name for aliyun csi disk provisioner
	AliyunCSIDiskProvisioner = "aliyun-csi-disk-provisioner"
	// AliyunCSINasPlugin name for aliyun csi nas plugin
	AliyunCSINasPlugin = "aliyun-csi-nas-plugin"
	// AliyunCSINasProvisioner name for aliyun csi nas provisioner
	AliyunCSINasProvisioner = "aliyun-csi-nas-provisioner"
)
