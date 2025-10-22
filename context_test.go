package pbjs_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/stevecallear/pbjs"
)

func TestGetContextHeader(t *testing.T) {
	hd := nats.Header{"key": []string{"value1", "value2"}}

	tests := []struct {
		name  string
		setup func(ctx context.Context) context.Context
		exp   nats.Header
		ok    bool
	}{
		{
			name: "should return an empty header if not set",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			exp: nats.Header{},
			ok:  false,
		},
		{
			name: "should return the header if set",
			setup: func(ctx context.Context) context.Context {
				return pbjs.SetContextHeader(ctx, hd)
			},
			exp: hd,
			ok:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup(context.Background())
			act := pbjs.GetContextHeader(ctx)

			if act, exp := act, tt.exp; !reflect.DeepEqual(act, exp) {
				t.Errorf("got %v, expected %v", act, exp)
			}
		})
	}
}

func TestGetContextMetadata(t *testing.T) {
	md := &jetstream.MsgMetadata{
		Sequence: jetstream.SequencePair{
			Consumer: 1,
			Stream:   2,
		},
	}

	tests := []struct {
		name  string
		setup func(ctx context.Context) context.Context
		exp   *jetstream.MsgMetadata
		ok    bool
	}{
		{
			name: "should return a empty metadata if not set",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			exp: &jetstream.MsgMetadata{},
			ok:  false,
		},
		{
			name: "should return the header if set",
			setup: func(ctx context.Context) context.Context {
				return pbjs.SetContextMetadata(ctx, md)
			},
			exp: md,
			ok:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setup(context.Background())
			act := pbjs.GetContextMetadata(ctx)

			if act, exp := act, tt.exp; !reflect.DeepEqual(act, exp) {
				t.Errorf("got %v, expected %v", act, exp)
			}
		})
	}
}
