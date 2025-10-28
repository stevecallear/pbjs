package pbjs_test

import (
	"errors"
	"testing"

	"github.com/stevecallear/pbjs"
)

func TestNewPersistentError(t *testing.T) {
	t.Run("should wrap the error", func(t *testing.T) {
		exp := errors.New("error")
		sut := pbjs.NewPersistentError(exp)
		act := sut.(interface{ Unwrap() error }).Unwrap()
		if act != exp {
			t.Errorf("got %v, expected %v", act, exp)
		}
	})

	t.Run("should return nil for nil error", func(t *testing.T) {
		act := pbjs.NewPersistentError(nil)
		if act != nil {
			t.Errorf("got %v, expected nil", act)
		}
	})
}

func TestIsPersistentError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		exp  bool
	}{
		{
			name: "should return true for persistent error",
			err:  pbjs.NewPersistentError(errors.New("error")),
			exp:  true,
		},
		{
			name: "should return false for other error",
			err:  errors.New("error"),
			exp:  false,
		},
		{
			name: "should return false for nil",
			err:  nil,
			exp:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act := pbjs.IsPersistentError(tt.err)
			if act != tt.exp {
				t.Errorf("got %v, expected %v", act, tt.exp)
			}
		})
	}
}

func TestPersistentError_Error(t *testing.T) {
	t.Run("should return the inner error", func(t *testing.T) {
		exp := errors.New("error")
		act := pbjs.NewPersistentError(exp)
		if act, exp := act.Error(), exp.Error(); act != exp {
			t.Errorf("got %s, expected %s", act, exp)
		}
	})
}

func TestPersistentError_Unwrap(t *testing.T) {
	t.Run("should return the inner error", func(t *testing.T) {
		exp := errors.New("error")
		sut := pbjs.NewPersistentError(exp)
		act := sut.(interface{ Unwrap() error }).Unwrap()
		if act != exp {
			t.Errorf("got %v, expected %v", act, exp)
		}
	})
}
