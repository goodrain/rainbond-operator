package constants

const (
	//Namespace namespace
	Namespace = "wt-system"
	//DefInstallPkgDestPath  Default destination path of the installation package extraction.
	DefInstallPkgDestPath = "/tmp/DefInstallPkgDestPath"
	//WutongClusterName wutong cluster resource name
	WutongClusterName = "wutongcluster"
	//WutongPackageName wutong package resource name
	WutongPackageName = "wutongpackage"
	// DefImageRepository is the default domain name of the mirror repository that Wutong is installed.
	DefImageRepository = "wutong.me"
	//GrDataPVC -
	GrDataPVC = "wt-cpt-grdata"
	// CachePVC -
	CachePVC = "wt-chaos-cache"
	// FoobarPVC -
	FoobarPVC = "foobar"
	// SpecialGatewayLabelKey is a special node label, used to specify where to install the wt-gateway
	SpecialGatewayLabelKey = "wutong.io/gateway"
	// SpecialChaosLabelKey is a special node label, used to specify where to install the wt-chaos
	SpecialChaosLabelKey = "wutong.io/chaos"
	// DefHTTPDomainSuffix -
	DefHTTPDomainSuffix = "grapps.cn"

	// AliyunCSIDiskPlugin name for aliyun csi disk plugin
	AliyunCSIDiskPlugin = "aliyun-csi-disk-plugin"
	// AliyunCSIDiskProvisioner name for aliyun csi disk provisioner
	AliyunCSIDiskProvisioner = "aliyun-csi-disk-provisioner"
	// AliyunCSINasPlugin name for aliyun csi nas plugin
	AliyunCSINasPlugin = "aliyun-csi-nas-plugin"
	// AliyunCSINasProvisioner name for aliyun csi nas provisioner
	AliyunCSINasProvisioner = "aliyun-csi-nas-provisioner"

	// ServiceAccountName is the name of service account
	ServiceAccountName = "wutong-operator"
)
