package model

type GlobalStatus string

const (
	// waiting waiting status
	Waiting GlobalStatus = "Waiting"
	// Initing initing status
	Initing GlobalStatus = "Initing"
	//Setting setting status
	Setting GlobalStatus = "Setting"
	//Installing installing status
	Installing GlobalStatus = "Installing"
	//Running running status
	Running GlobalStatus = "Running"
	//UnInstalling uninstalling status
	UnInstalling GlobalStatus = "UnInstalling"
)

const (
	// SuccessMessage success message
	SuccessMessage = "success"
	// HelpMessage help message
	HelpMessage = "internal error, please contact rainbond for help"
	// CantUpdateConfig can't update config
	CantUpdateConfig = "current cluster status can't update config"
	//IllegalConfigData illegal config data
	IllegalConfigData = "illegal config parameter, can't parse normally"
	// IllegalConfigDataFormat custom illegal config data
	IllegalConfigDataFormat = "%s parameter illegal, please modify it and retry"
)

// ReturnStruct return struct
type ReturnStruct struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// ReturnStructWithData return struct with data
type ReturnStructWithData struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}
