package model

type GlobalStatus string

const (
	// 等待中
	Waiting GlobalStatus = "Waiting"
	// 初始化中
	Initing GlobalStatus = "Initing"
	//配置中
	Setting GlobalStatus = "Setting"
	//安装中
	Installing GlobalStatus = "Installing"
	//运行中
	Running GlobalStatus = "Running"
	//卸载中
	UnInstalling GlobalStatus = "UnInstalling"
)
