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
