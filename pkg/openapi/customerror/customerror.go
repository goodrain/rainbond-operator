package customerror

// DownLoadError download errorr
type DownLoadError struct {
	Msg  string
	Code int
}

// NewDownLoadError new dowload error
func NewDownLoadError(msg string) *DownLoadError {
	return &DownLoadError{Code: 1001, Msg: msg}
}

// Error error
func (err *DownLoadError) Error() string {
	return err.Msg
}

// DownloadingError downloading error
type DownloadingError struct {
	Msg  string
	Code int
}

// Error error
func (err *DownloadingError) Error() string {
	return err.Msg
}

// NewDownloadingError new downloading error
func NewDownloadingError(msg string) *DownloadingError {
	return &DownloadingError{Code: 1002, Msg: msg}
}

// RainbondTarNotExistError RainbondTarNotExistError
type RainbondTarNotExistError struct {
	Msg  string
	Code int
}

// Error error
func (err *RainbondTarNotExistError) Error() string {
	return err.Msg
}

// NewRainbondTarNotExistError new rainbond tar not exist error
func NewRainbondTarNotExistError(msg string) *RainbondTarNotExistError {
	return &RainbondTarNotExistError{Code: 1003, Msg: msg}
}

// CRNotFoundError rbd cr not found error
type CRNotFoundError struct {
	Msg  string
	Code int
}

// Error error
func (err *CRNotFoundError) Error() string {
	return err.Msg
}

// NewCRNotFoundError new rbd cr not found error
func NewCRNotFoundError(msg string) *CRNotFoundError {
	return &CRNotFoundError{Code: 1004, Msg: msg}
}
