package handler

const (
	// rainbondVolumeNotFound -
	rainbondVolumeNotFound = "rainbond volume not found"
)

type IgnoreError struct {
	msg string
}

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

func IsRainbondVolumeNotFound(e error) bool {
	err, ok := e.(*IgnoreError)
	if !ok {
		return false
	}
	return err.msg == rainbondVolumeNotFound
}
