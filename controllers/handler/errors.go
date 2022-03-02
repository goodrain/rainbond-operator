package handler

const (
	// WutongVolumeNotFound -
	WutongVolumeNotFound = "wutong volume not found"
)

// IgnoreError is the error with ignore by WutongComponent controller.
type IgnoreError struct {
	msg string
}

// NewIgnoreError creates a new IgnoreError
func NewIgnoreError(msg string) *IgnoreError {
	return &IgnoreError{msg: msg}
}

func (i *IgnoreError) Error() string {
	return i.msg
}

// IsIgnoreError check if the given err is IgnoreError.
func IsIgnoreError(err error) bool {
	_, ok := err.(*IgnoreError)
	return ok
}

// IsWutongVolumeNotFound checks if the given error is WutongVolumeNotFound.
func IsWutongVolumeNotFound(e error) bool {
	err, ok := e.(*IgnoreError)
	if !ok {
		return false
	}
	return err.msg == WutongVolumeNotFound
}
