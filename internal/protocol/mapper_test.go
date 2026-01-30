package protocol_test

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/labi-le/belphegor/internal/protocol"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/internal/types/proto"
	"github.com/labi-le/belphegor/pkg/mime"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	testTime = time.Now()

	fullMsgEvent = domain.EventMessage{
		From:    domain.NodeID(101),
		Created: testTime,
		Payload: domain.Message{
			ID:            domain.MessageID(102),
			Data:          []byte{0xDE, 0xAD},
			MimeType:      mime.TypeImage,
			ContentHash:   0xCAFEBABE,
			ContentLength: 1024,
			Name:          "image.png",
		},
	}

	fullAnnEvent = domain.EventAnnounce{
		From:    domain.NodeID(201),
		Created: testTime,
		Payload: domain.Announce{
			ID:            domain.MessageID(202),
			MimeType:      mime.TypePath,
			ContentHash:   0xDEADBEEF,
			ContentLength: 2048,
		},
	}

	fullReqEvent = domain.EventRequest{
		From:    domain.NodeID(301),
		Created: testTime,
		Payload: domain.Request{
			ID: domain.MessageID(302),
		},
	}

	fullHandshakeEvent = domain.EventHandshake{
		From:    0,
		Created: testTime,
		Payload: domain.Handshake{
			Version: "1.2.3",
			Port:    8080,
			MetaData: domain.Device{
				ID:   domain.NodeID(401),
				Name: "TestNode",
				Arch: "amd64",
			},
		},
	}
)

func TestMapping_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		in   any
	}{
		{"EventMessage", fullMsgEvent},
		{"EventAnnounce", fullAnnEvent},
		{"EventRequest", fullReqEvent},
		{"EventHandshake", fullHandshakeEvent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := protocol.MustEncode(tt.in)

			decoded, err := protocol.DecodeEvent(bytes.NewReader(encoded))
			if err != nil {
				t.Fatalf("DecodeEvent failed: %v", err)
			}

			opts := []cmp.Option{
				cmpopts.EquateApproxTime(time.Microsecond),

				cmpopts.IgnoreFields(domain.Message{}, "Data"),

				cmpopts.IgnoreFields(domain.EventMessage{}, "From"),
				cmpopts.IgnoreFields(domain.EventAnnounce{}, "From"),
				cmpopts.IgnoreFields(domain.EventRequest{}, "From"),
			}

			if diff := cmp.Diff(tt.in, decoded, opts...); diff != "" {
				t.Errorf("RoundTrip mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestProto_Completeness(t *testing.T) {
	tests := []struct {
		name string
		in   any
	}{
		{"EventMessage", fullMsgEvent},
		{"EventAnnounce", fullAnnEvent},
		{"EventRequest", fullReqEvent},
		{"EventHandshake", fullHandshakeEvent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := protocol.MapToProto(tt.in)
			if pb == nil {
				t.Fatalf("MapToProto returned nil for %T", tt.in)
			}

			assertNonZero(t, pb, []string{})

			payload := getProtoPayload(pb)
			if payload == nil {
				t.Fatal("Proto payload is nil")
			}
			assertNonZero(t, payload, []string{})
		})
	}
}

func assertNonZero(t *testing.T, v interface{}, parentPath []string) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		structField := typ.Field(i)
		fieldName := structField.Name

		if !structField.IsExported() || (len(fieldName) > 4 && fieldName[:4] == "XXX_") {
			continue
		}

		currentPath := append(parentPath, fieldName)
		pathStr := joinPath(currentPath)

		if field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			if ts, ok := field.Interface().(*timestamppb.Timestamp); ok {
				if ts.Seconds == 0 && ts.Nanos == 0 {
					t.Errorf("Field '%s' is zero (Timestamp)", pathStr)
				}
				continue
			}
			assertNonZero(t, field.Interface(), currentPath)
			continue
		}

		if isZero(field) {
			t.Errorf("Field '%s' is zero/empty. Missing mapping in MapToProto?", pathStr)
		}
	}
}

func getProtoPayload(event *proto.Event) interface{} {
	switch p := event.Payload.(type) {
	case *proto.Event_Message:
		return p.Message
	case *proto.Event_Announce:
		return p.Announce
	case *proto.Event_Request:
		return p.Request
	case *proto.Event_Handshake:
		return p.Handshake
	default:
		return nil
	}
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Struct:
		return false
	default:
		return v.IsZero()
	}
}

func joinPath(p []string) string {
	res := ""
	for i, s := range p {
		if i > 0 {
			res += "."
		}
		res += s
	}
	return res
}
