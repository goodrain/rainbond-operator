package translate

import "os"

var translationMetadata = map[string]string{
	"download rainbond.tar error, please try again or upload it using /uploads": "下载rainbond安装包失败，请重试或者使用上传接口(/uploads)上传",
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
