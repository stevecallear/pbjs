package pbjs_test

import (
	"errors"
	"testing"

	"github.com/stevecallear/pbjs"
)

func TestErrorTypeOf(t *testing.T) {
	err := errors.New("error")
	tests := []struct {
		name string
		err  error
		exp  pbjs.ErrorType
	}{
		{
			name: "should return the error type",
			err:  pbjs.NewError(err, pbjs.ErrorTypePersistent),
			exp:  pbjs.ErrorTypePersistent,
		},
		{
			name: "should return unknown if no type",
			err:  err,
			exp:  pbjs.ErrorTypeUnknown,
		},
		{
			name: "should return none if nil error",
			err:  nil,
			exp:  pbjs.ErrorTypeNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act := pbjs.ErrorTypeOf(tt.err)
			if act != tt.exp {
				t.Errorf("got %v, expected %v", act, tt.exp)
			}
		})
	}
}

func TestNewError(t *testing.T) {
	t.Run("should return nil for nil error", func(t *testing.T) {
		err := pbjs.NewError(nil, pbjs.ErrorTypePersistent)
		if err != nil {
			t.Errorf("got %v, expected nil", err)
		}
	})
}

func TestError_Error(t *testing.T) {
	t.Run("should return the inner error", func(t *testing.T) {
		const exp = "error"
		err := pbjs.NewError(errors.New(exp), pbjs.ErrorTypeTransient)
		act := err.Error()
		if act != exp {
			t.Errorf("got %s, expected %s", act, exp)
		}
	})
}

func TestError_Unwrap(t *testing.T) {
	t.Run("should return the inner error", func(t *testing.T) {
		exp := errors.New("error")
		err := pbjs.NewError(exp, pbjs.ErrorTypeTransient)
		act := err.(*pbjs.Error).Unwrap()
		if act != exp {
			t.Errorf("got %v, expected %v", act, exp)
		}
	})
}
