package secrets

import "fmt"

// Error represents a failure with a specific exit code.
type Error struct {
	Code int
	Err  error
}

// Error codes aligned with CLI contract.
const (
	ErrCodeValidation = 70
	ErrCodeEncryption = 71
)

// NewError constructs an Error wrapper.
func NewError(code int, err error) *Error {
	return &Error{Code: code, Err: err}
}

func (e *Error) Error() string {
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) String() string {
	return fmt.Sprintf("code=%d err=%v", e.Code, e.Err)
}
