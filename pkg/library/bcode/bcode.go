package bcode

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/pkg/errors"
)

// Coder has ability to get code, msg or detail from error.
type Coder interface {
	// http status code
	Status() int
	// business code
	Code() int
	Error() string
	Msg() string
}

var (
	codes    = make(map[int]struct{})
	messages atomic.Value
)

func newCode(i int) Coder {
	if _, ok := codes[i]; ok {
		panic(fmt.Sprintf("bcode %d already exists", i))
	}
	codes[i] = struct{}{}
	return Code{code: i}
}

func newCodeWithMsg(i int, msg string) Coder {
	if _, ok := codes[i]; ok {
		panic(fmt.Sprintf("bcode %d already exists", i))
	}
	codes[i] = struct{}{}
	return Code{code: i, msg: msg}
}

func newCodeWithStatus(s, i int, msg string) Coder {
	if _, ok := codes[i]; ok {
		panic(fmt.Sprintf("bcode %d already exists", i))
	}
	if s < 100 || s > 599 {
		panic(fmt.Sprintf("invalid http status code: %d", s))
	}
	codes[i] = struct{}{}
	return Code{status: s, code: i, msg: msg}
}

// Code represents a business code.
type Code struct {
	status, code int
	msg          string
}

// Status returns a http status code.
func (c Code) Status() int {
	if c.code == 0 {
		return 200
	}
	if c.code >= 100 && c.code <= 599 {
		return c.code
	}
	return c.status
}

// Code returns a business code
func (c Code) Code() int {
	return c.code
}

func (c Code) Error() string {
	return strconv.FormatInt(int64(c.code), 10)
}

// Msg returns a message
func (c Code) Msg() string {
	if c.msg != "" {
		return c.msg
	}
	return c.Error()
}

// Err2Coder converts the given err to Coder.
func Err2Coder(err error) Coder {
	if err == nil {
		return OK
	}
	coder, ok := errors.Cause(err).(Coder)
	if ok {
		return coder
	}
	return Str2Coder(err.Error())
}

// Str2Coder converts the given str to Coder.
func Str2Coder(str string) Coder {
	str = strings.TrimSpace(str)
	if str == "" {
		return OK
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		return ServerErr
	}
	c := Code{
		code: i,
	}
	return c
}
