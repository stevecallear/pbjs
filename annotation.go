package pbjs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

	pbjspb "github.com/stevecallear/pbjs/proto/pbjs"
)

type messageOption struct {
	subject   string
	templated bool
}

var (
	messageOptionCache   = map[string]messageOption{}
	messageOptionCacheMu = new(sync.RWMutex)
)

// AnnotationSubjectConvention returns a subject for the message based on proto annotations.
// If the required annotation is not present on the message then an error will be returned.
func AnnotationSubjectConvention(ctx context.Context, m proto.Message) (string, error) {
	fn := string(m.ProtoReflect().Descriptor().FullName())

	mo, ok := func() (messageOption, bool) {
		messageOptionCacheMu.RLock()
		defer messageOptionCacheMu.RUnlock()
		o, ok := messageOptionCache[fn]
		return o, ok
	}()
	if ok {
		if mo.templated {
			return ExecuteSubjectTemplate(mo.subject, m)
		}
		return mo.subject, nil
	}

	messageOptionCacheMu.Lock()
	defer messageOptionCacheMu.Unlock()

	if o, ok := messageOptionCache[fn]; ok {
		if o.templated {
			return ExecuteSubjectTemplate(o.subject, m)
		}
		return o.subject, nil
	}

	md, _ := m.ProtoReflect().Descriptor().Options().(*descriptorpb.MessageOptions)
	if md == nil {
		return "", errors.New("message options not defined")
	}

	pbo, _ := proto.GetExtension(md, pbjspb.E_Message).(*pbjspb.Message)
	mo = messageOption{subject: pbo.GetSubject()}

	var sub string
	var err error
	sub, mo.templated, err = executeSubjectTemplate(mo.subject, m)
	if err != nil {
		return "", err
	}

	messageOptionCache[fn] = mo
	return sub, nil
}

// ExecuteSubjectTemplate replaces field identifiers with their message values
func ExecuteSubjectTemplate(template string, m proto.Message) (string, error) {
	sub, _, err := executeSubjectTemplate(template, m)
	return sub, err
}

func executeSubjectTemplate(template string, m proto.Message) (_ string, templated bool, _ error) {
	if template == "" {
		return "", false, errors.New("empty template")
	}

	var builder strings.Builder
	builder.Grow(len(template) * 2)

	var last int
	for {
		open := strings.Index(template[last:], "{")
		if open == -1 {
			builder.WriteString(template[last:])
			break
		}

		open += last
		close := strings.Index(template[open+1:], "}")
		if close == -1 {
			return "", true, fmt.Errorf("no closing brace after %d", open)
		}

		close += open + 1
		builder.WriteString(template[last:open])

		field := template[open+1 : close]
		descriptor := m.ProtoReflect().Descriptor().Fields().ByTextName(field)
		if descriptor == nil {
			return "", true, fmt.Errorf("invalid field: %s", field)
		}

		value := m.ProtoReflect().Get(descriptor).String()
		if value == "" {
			return "", true, fmt.Errorf("empty field value: %s", field)
		}
		builder.WriteString(value)

		last = close + 1
	}
	return builder.String(), last > 0, nil
}
