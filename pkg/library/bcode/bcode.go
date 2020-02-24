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

// Code represents a business code.
type Code struct {
	code int
	msg  string
}

func (c Code) Code() int {
	return c.code
}

func (c Code) Error() string {
	return strconv.FormatInt(int64(c.code), 10)
}

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
