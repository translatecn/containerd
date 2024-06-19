package errors

import (
	"fmt"
	"io"
	"net/http"
)

var _ error = ErrUnexpectedStatus{}

// ErrUnexpectedStatus is returned if a registry API request returned with unexpected HTTP status
type ErrUnexpectedStatus struct {
	Status                    string
	StatusCode                int
	Body                      []byte
	RequestURL, RequestMethod string
}

func (e ErrUnexpectedStatus) Error() string {
	return fmt.Sprintf("unexpected status from %s request to %s: %s", e.RequestMethod, e.RequestURL, e.Status)
}

// NewUnexpectedStatusErr creates an ErrUnexpectedStatus from HTTP response
func NewUnexpectedStatusErr(resp *http.Response) error {
	var b []byte
	if resp.Body != nil {
		b, _ = io.ReadAll(io.LimitReader(resp.Body, 64000)) // 64KB
	}
	err := ErrUnexpectedStatus{
		Body:          b,
		Status:        resp.Status,
		StatusCode:    resp.StatusCode,
		RequestMethod: resp.Request.Method,
	}
	if resp.Request.URL != nil {
		err.RequestURL = resp.Request.URL.String()
	}
	return err
}
