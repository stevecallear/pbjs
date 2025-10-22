package pbjs

import "errors"

type (
	// Error represents a handler error
	Error struct {
		typ ErrorType
		err error
	}

	// ErrorType indicates an error type
	ErrorType uint8
)

const (
	// ErrorTypeNone indicates that there is no error.
	// It is returned only when a nil error is supplied to [ErrorTypeOf].
	ErrorTypeNone ErrorType = iota

	// ErrorTypeUnknown indicates an unknown error.
	// It is returned only when the error supplied to [ErrorTypeOf] does not have a type.
	ErrorTypeUnknown

	// ErrorTypeTransient indicates that the handler error was transient.
	// By default transient errors result in Nak.
	ErrorTypeTransient

	// ErrorTypePersistent indicates that the handler error was persistent.
	// By default persistent errors result in Term.
	ErrorTypePersistent
)

// ErrorTypeOf returns the [ErrorType] of the supplied error
func ErrorTypeOf(err error) ErrorType {
	if err == nil {
		return ErrorTypeNone
	}

	var e *Error
	if errors.As(err, &e) {
		return e.typ
	}
	return ErrorTypeUnknown
}

// NewError wraps the supplied error with the specified types
func NewError(err error, typ ErrorType) error {
	if err == nil {
		return nil
	}

	return &Error{
		typ: typ,
		err: err,
	}
}

// Error returns the inner error message
func (e *Error) Error() string {
	return e.err.Error()
}

// Unwrap returns the inner error
func (e *Error) Unwrap() error {
	return e.err
}
