package error

import "fmt"

type ParamError struct {
	ParamName string
}

type WriteError struct {
	Err error
	Src string
}

type ReadError struct {
	Err error
	Src string
}

func (e ParamError) Error() string {
	return fmt.Sprintf("%s is required", e.ParamName)
}

func (e WriteError) Error() string {
	return "Error while writing into " + e.Src + e.Err.Error()
}

func (e ReadError) Error() string {
	return "Error while reading " + e.Src + e.Err.Error()
}
