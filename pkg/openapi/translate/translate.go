package translate

import "os"

var translationMetadata = map[string]string{
	"download rainbond.tar error, please try again or upload it using /uploads": "下载rainbond安装包失败，请重试或者使用上传接口(/uploads)上传",
	"cluster is installing, can't update config":                                "集群正在安装中，暂不可修改配置",
	"step_setting":           "配置集群环境",
	"step_download":          "下载安装包",
	"step_prepare_storage":   "准备存储",
	"step_prepare_image_hub": "准备镜像仓库",
	"step_unpacke":           "解压安装包",
	"step_load_image":        "加载镜像",
	"step_push_image":        "上传镜像",
	"step_install_component": "安装基础服务",
	"status_waiting":         "等待中",
	"status_processing":      "进行中",
	"status_finished":        "已完成",
	"status_failed":          "失败",
}

//Translation Translation English to Chinese
func Translation(english string) string {
	if chinese, ok := translationMetadata[english]; ok {
		if os.Getenv("RAINBOND_LANG") == "en" {
			return english
		}
		return chinese
	}
	return english
}
