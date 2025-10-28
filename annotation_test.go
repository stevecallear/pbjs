package pbjs_test

import (
	"testing"

	"github.com/stevecallear/pbjs"
	"github.com/stevecallear/pbjs/internal/proto/testpb"
	"google.golang.org/protobuf/proto"
)

func BenchmarkAnnotationSubjectConvention_Subject(b *testing.B) {
	m := new(testpb.DispatchOrderCommand)
	for b.Loop() {
		_, err := pbjs.AnnotationSubjectConvention(b.Context(), m)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAnnotationSubjectConvention_Template(b *testing.B) {
	m := &testpb.OrderDispatchedEvent{Id: "abc123"}
	for b.Loop() {
		_, err := pbjs.AnnotationSubjectConvention(b.Context(), m)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestAnnotationSubjectConvention(t *testing.T) {
	tests := []struct {
		name string
		msg  proto.Message
		exp  string
		err  bool
	}{
		{
			name: "should apply the raw subject",
			msg:  &testpb.DispatchOrderCommand{},
			exp:  "CMD.ORDER.dispatch",
		},
		{
			name: "should apply the template",
			msg:  &testpb.OrderDispatchedEvent{Id: "abc123"},
			exp:  "ORDER.abc123.dispatched",
		},
		{
			name: "should support basic types at root level",
			msg: &testpb.TemplateTypes{
				S:   "str",
				I32: 1,
				I64: 2,
				F32: 3.3,
				F64: 4.4,
				B:   true,
			},
			exp: "s_str.i32_1.i64_2.f32_3.3.f64_4.4.b_true",
		},
		{
			name: "should return an error if the field is empty",
			msg:  &testpb.OrderDispatchedEvent{},
			err:  true,
		},
		{
			name: "should return an error if the annotation is missing",
			msg:  &testpb.MissingAnnotation{},
			err:  true,
		},
		{
			name: "should return an error if the annotation is invalid",
			msg:  &testpb.InvalidAnnotation{},
			err:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 2 { // run all cases twice to ensure correct cache behaviour
				act, err := pbjs.AnnotationSubjectConvention(t.Context(), tt.msg)
				if err != nil && !tt.err {
					t.Errorf("got %v, expected nil", err)
				}
				if err == nil && tt.err {
					t.Error("got nil, expected error")
				}
				if act != tt.exp {
					t.Errorf("got %s, expected %s", act, tt.exp)
				}
			}
		})
	}
}

func TestExecuteSubjectTemplate(t *testing.T) {
	tests := []struct {
		name     string
		msg      proto.Message
		template string
		exp      string
		err      bool
	}{
		{
			name: "should apply the template",
			msg: &testpb.TemplateTypes{
				S:   "str",
				I32: 1,
				I64: 2,
				F32: 3.3,
				F64: 4.4,
				B:   true,
			},
			template: "s_{s}.i32_{i32}.i64_{i64}.f32_{f32}.f64_{f64}.b_{b}",
			exp:      "s_str.i32_1.i64_2.f32_3.3.f64_4.4.b_true",
		},
		{
			name:     "should handle untemplated subjects",
			msg:      &testpb.DispatchOrderCommand{},
			template: "CMD.ORDER.dispatch",
			exp:      "CMD.ORDER.dispatch",
		},
		{
			name:     "should handle field only",
			msg:      &testpb.OrderDispatchedEvent{Id: "abc123"},
			template: "{id}",
			exp:      "abc123",
		},
		{
			name:     "should handle template suffix",
			msg:      &testpb.OrderDispatchedEvent{Id: "abc123"},
			template: "{id}.dispatched",
			exp:      "abc123.dispatched",
		},
		{
			name:     "should return error on empty field",
			msg:      &testpb.OrderDispatchedEvent{},
			template: "{id}",
			err:      true,
		},
		{
			name:     "should return an error on invalid syntax 2",
			template: "{{",
			err:      true,
		},
		{
			name:     "should return an error on invalid syntax 3",
			msg:      &testpb.OrderDispatchedEvent{},
			template: "{}",
			err:      true,
		},
		{
			name:     "should return an error on invalid field",
			template: "{other}",
			msg:      &testpb.TemplateTypes{},
			err:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act, err := pbjs.ExecuteSubjectTemplate(tt.template, tt.msg)
			if err != nil && !tt.err {
				t.Errorf("got %v, expected nil", err)
			}
			if err == nil && tt.err {
				t.Error("got nil, expected error")
			}
			if act != tt.exp {
				t.Errorf("got %s, expected %s", act, tt.exp)
			}
		})
	}
}
