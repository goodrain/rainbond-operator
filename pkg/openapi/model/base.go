package model

// GlobalStatus -
type GlobalStatus string

const (
	// Waiting waiting waiting status
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
