package service

import "fmt"

type Error struct {
	Code int
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

func bad(msg string) error  { return &Error{Code: 400, Msg: msg} }
func miss(msg string) error { return &Error{Code: 404, Msg: msg} }
func deny(msg string) error { return &Error{Code: 409, Msg: msg} }

func wrapBad(prefix string, err error) error {
	if err == nil {
		return nil
	}
	return bad(fmt.Sprintf("%s: %s", prefix, err.Error()))
}
