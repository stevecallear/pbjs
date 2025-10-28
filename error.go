package pbjs

import "errors"

type persistentError struct {
	err error
}

// NewPersistentError wraps the supplied error to indicate that it is persistent
func NewPersistentError(err error) error {
	if err == nil {
		return nil
	}
	return &persistentError{err: err}
}

// IsPersistentError returns true if the supplied error is persistent
func IsPersistentError(err error) bool {
	var pe *persistentError
	return errors.As(err, &pe)
}

// Error returns the inner error message
func (e *persistentError) Error() string {
	return e.err.Error()
}

// Unwrap returns the inner error
func (e *persistentError) Unwrap() error {
	return e.err
}
