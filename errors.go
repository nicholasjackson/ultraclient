package loadbalancer

import (
	"bytes"
	"fmt"
)

const (
	ErrorTimeout                 = "timeout"
	ErrorCircuitOpen             = "circuit open"
	ErrorGeneral                 = "general error"
	ErrorUnableToCompleteRequest = "unable to complete request"
)

type ClientError struct {
	errors []error
}

func (s *ClientError) AddError(err error) {
	s.errors = append(s.errors, err)
}

func (s *ClientError) Errors() []error {
	return s.errors
}

// Error implements the error interface
func (s ClientError) Error() string {
	writer := bytes.NewBufferString("")
	for _, err := range s.errors {
		fmt.Fprint(writer, err.Error())
	}

	return writer.String()
}
