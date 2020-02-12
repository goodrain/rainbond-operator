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

func (err *RainbondTarNotExistError) Error() string {
	return err.Msg
}

// NewRainbondTarNotExistError new rainbond tar not exist error
func NewRainbondTarNotExistError(msg string) *RainbondTarNotExistError {
	return &RainbondTarNotExistError{Code: 1003, Msg: msg}
}

//CRAlreadyExistsError cr already exists error
type CRAlreadyExistsError struct {
	Msg  string
	Code int
}

// Error error
func (err *CRAlreadyExistsError) Error() string {
	return err.Msg
}

// NewCRAlreadyExistsError new cr already exists error
func NewCRAlreadyExistsError(msg string) *CRAlreadyExistsError {
	return &CRAlreadyExistsError{
		Msg:  msg,
		Code: 1004,
	}
}
