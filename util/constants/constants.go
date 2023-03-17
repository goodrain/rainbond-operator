package constants

const (
	// WutongSystemNamespace wt-system
	WutongSystemNamespace = "wt-system"
	// DefInstallPkgDestPath  Default destination path of the installation package extraction.
	DefInstallPkgDestPath = "/tmp/DefInstallPkgDestPath"
	// WutongClusterName wutong cluster resource name
	WutongClusterName = "wutongcluster"
	// WutongPackageName wutong package resource name
	WutongPackageName = "wutongpackage"
	// DefImageRepository is the default domain name of the mirror repository that Wutong is installed.
	DefImageRepository = "wutong.me"
	// WTDataPVC -
	WTDataPVC = "wt-cpt-wtdata"
	// CachePVC -
	CachePVC = "wt-chaos-cache"
	// FoobarPVC -
	// FoobarPVC = "foobar"
	// SpecialGatewayLabelKey is a special node label, used to specify where to install the wt-gateway
	SpecialGatewayLabelKey = "wutong.io/gateway"
	// SpecialChaosLabelKey is a special node label, used to specify where to install the wt-chaos
	SpecialChaosLabelKey = "wutong.io/chaos"
	// DefHTTPDomainSuffix -
	DefHTTPDomainSuffix = "wtapps.cn"

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

	// InstallImageRepo install image repo
	InstallImageRepo = "swr.cn-southwest-2.myhuaweicloud.com/wutong"
)
