package gitlab

import "fmt"

// SystemFailureError signals GitLab Runner to treat the failure as a system
// failure (retry prepare using SYSTEM_FAILURE_EXIT_CODE).
type SystemFailureError struct {
	Err error
}

func NewSystemFailureError(err error) *SystemFailureError {
	return &SystemFailureError{Err: err}
}

func (e *SystemFailureError) Error() string {
	return fmt.Sprintf("system failure: %v", e.Err)
}

func (e *SystemFailureError) Unwrap() error {
	return e.Err
}
