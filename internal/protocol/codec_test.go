package protocol_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/labi-le/belphegor/internal/protocol"
	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/mime"
)

// event fixtures (fullMsgEvent, fullAnnEvent, ...) are declared in mapper_test.go.
var codecEvents = []struct {
	name  string
	event any
}{
	{"message", fullMsgEvent},
	{"announce", fullAnnEvent},
	{"request", fullReqEvent},
	{"handshake", fullHandshakeEvent},
}

// BenchmarkEncode measures the full outbound path: domain->proto mapping,
// protobuf marshal, and pool recycling, per event type.
func BenchmarkEncode(b *testing.B) {
	for _, bm := range codecEvents {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				if _, err := protocol.Encode(bm.event); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkDecodeEvent measures the full inbound path: protobuf unmarshal and
// proto->domain mapping, per event type.
func BenchmarkDecodeEvent(b *testing.B) {
	for _, bm := range codecEvents {
		encoded := protocol.MustEncode(bm.event)
		b.Run(bm.name, func(b *testing.B) {
			r := bytes.NewReader(encoded)
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				r.Reset(encoded)
				if _, err := protocol.DecodeEvent(r); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkWriteEvent measures the true per-message send hot path used by peers
// (WriteContext -> WriteEvent -> pooled EncodeToWriter); the marshal buffer is
// pooled, so residual allocations come only from the domain->proto mapping.
func BenchmarkWriteEvent(b *testing.B) {
	for _, bm := range codecEvents {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for range b.N {
				if err := protocol.WriteEvent(io.Discard, bm.event); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestCodec_PoolReuse_VaryingSizes guards the pooled Event/Timestamp reuse: it
// encodes+decodes events with different timestamp varint widths and name lengths
// back-to-back across several rounds. A stale cached size or reused-timestamp bug
// would corrupt the wire bytes and fail the round-trip.
func TestCodec_PoolReuse_VaryingSizes(t *testing.T) {
	cases := []domain.EventMessage{
		{From: 1, Created: time.Unix(1, 1), Payload: domain.Message{ID: 1, MimeType: mime.TypeText, ContentHash: 1, ContentLength: 1, Name: "a"}},
		{From: 2, Created: time.Unix(1<<40, 999_999_999), Payload: domain.Message{ID: 2, MimeType: mime.TypeImage, ContentHash: 9, ContentLength: 9, Name: "a-much-longer-name-that-changes-varint-size"}},
	}

	for round := range 5 {
		for _, want := range cases {
			enc := protocol.MustEncode(want)
			got, err := protocol.DecodeExpect[domain.EventMessage](bytes.NewReader(enc))
			if err != nil {
				t.Fatalf("round %d: decode: %v", round, err)
			}
			if got.Payload.ID != want.Payload.ID || got.Payload.Name != want.Payload.Name {
				t.Fatalf("round %d: payload mismatch: got %+v want %+v", round, got.Payload, want.Payload)
			}
			if !got.Created.Equal(want.Created) {
				t.Fatalf("round %d: created mismatch: got %v want %v", round, got.Created, want.Created)
			}
		}
	}
}

type unsupportedEvent struct{}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrShortWrite }

// TestWriteEvent_RoundTrip locks the send path (WriteEvent -> pooled
// EncodeToWriter) against the receive path (DecodeExpect -> DecodeReader): a
// length-prefixed frame must decode back to the same typed event.
func TestWriteEvent_RoundTrip(t *testing.T) {
	t.Run("message", func(t *testing.T) {
		var buf bytes.Buffer
		if err := protocol.WriteEvent(&buf, fullMsgEvent); err != nil {
			t.Fatalf("WriteEvent: %v", err)
		}
		got, err := protocol.DecodeExpect[domain.EventMessage](&buf)
		if err != nil {
			t.Fatalf("DecodeExpect: %v", err)
		}
		if got.Payload.ID != fullMsgEvent.Payload.ID || got.Payload.MimeType != fullMsgEvent.Payload.MimeType {
			t.Errorf("round-trip mismatch: got ID=%d mime=%v", got.Payload.ID, got.Payload.MimeType)
		}
	})

	t.Run("handshake", func(t *testing.T) {
		var buf bytes.Buffer
		if err := protocol.WriteEvent(&buf, fullHandshakeEvent); err != nil {
			t.Fatalf("WriteEvent: %v", err)
		}
		got, err := protocol.DecodeExpect[domain.EventHandshake](&buf)
		if err != nil {
			t.Fatalf("DecodeExpect: %v", err)
		}
		if got.Payload.Version != fullHandshakeEvent.Payload.Version || got.Payload.Port != fullHandshakeEvent.Payload.Port {
			t.Errorf("round-trip mismatch: %+v", got.Payload)
		}
	})
}

// TestWriteEvent_SequentialFraming guards the 4-byte length prefix: two events
// written back-to-back must read back independently and in order.
func TestWriteEvent_SequentialFraming(t *testing.T) {
	var buf bytes.Buffer
	if err := protocol.WriteEvent(&buf, fullReqEvent); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if err := protocol.WriteEvent(&buf, fullHandshakeEvent); err != nil {
		t.Fatalf("write second: %v", err)
	}

	req, err := protocol.DecodeExpect[domain.EventRequest](&buf)
	if err != nil {
		t.Fatalf("decode first: %v", err)
	}
	if req.Payload.ID != fullReqEvent.Payload.ID {
		t.Errorf("first frame ID = %d, want %d", req.Payload.ID, fullReqEvent.Payload.ID)
	}

	hs, err := protocol.DecodeExpect[domain.EventHandshake](&buf)
	if err != nil {
		t.Fatalf("decode second: %v", err)
	}
	if hs.Payload.Port != fullHandshakeEvent.Payload.Port {
		t.Errorf("second frame port = %d, want %d", hs.Payload.Port, fullHandshakeEvent.Payload.Port)
	}
}

// TestWriteEvent_WriterError ensures a failing writer surfaces an error instead
// of being silently dropped.
func TestWriteEvent_WriterError(t *testing.T) {
	if err := protocol.WriteEvent(failingWriter{}, fullMsgEvent); err == nil {
		t.Fatal("WriteEvent must propagate writer errors")
	}
}

// TestEncode_UnsupportedType covers the guard rejecting non-event values.
func TestEncode_UnsupportedType(t *testing.T) {
	if _, err := protocol.Encode(unsupportedEvent{}); err == nil {
		t.Fatal("Encode must reject an unsupported type")
	}
}

// TestWriteEvent_UnsupportedType ensures nothing is written for a bad type.
func TestWriteEvent_UnsupportedType(t *testing.T) {
	var buf bytes.Buffer
	if err := protocol.WriteEvent(&buf, unsupportedEvent{}); err == nil {
		t.Fatal("WriteEvent must reject an unsupported type")
	}
	if buf.Len() != 0 {
		t.Errorf("wrote %d bytes for an unsupported type, want 0", buf.Len())
	}
}

// TestMustEncode_PanicsOnUnsupported locks the panic contract of MustEncode.
func TestMustEncode_PanicsOnUnsupported(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("MustEncode must panic on an unsupported type")
		}
	}()
	_ = protocol.MustEncode(unsupportedEvent{})
}

// TestDecodeEvent_InvalidInput covers decode error paths: truncated frame
// header, a length prefix longer than the body, and an empty stream.
func TestDecodeEvent_InvalidInput(t *testing.T) {
	cases := map[string][]byte{
		"truncated header":         {0x01},
		"body shorter than length": {0x00, 0x00, 0x00, 0x10, 0x01, 0x02},
		"empty":                    nil,
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := protocol.DecodeEvent(bytes.NewReader(in)); err == nil {
				t.Fatal("DecodeEvent must return an error")
			}
		})
	}
}

// TestDecodeExpect_TypeMismatch ensures the typed helper rejects a decoded event
// of the wrong concrete type instead of returning a zero value.
func TestDecodeExpect_TypeMismatch(t *testing.T) {
	var buf bytes.Buffer
	if err := protocol.WriteEvent(&buf, fullMsgEvent); err != nil {
		t.Fatal(err)
	}
	if _, err := protocol.DecodeExpect[domain.EventAnnounce](&buf); err == nil {
		t.Fatal("DecodeExpect must error when the decoded type differs")
	}
}

// TestDecodeExpect_PropagatesDecodeError ensures underlying decode failures
// surface through the typed helper.
func TestDecodeExpect_PropagatesDecodeError(t *testing.T) {
	if _, err := protocol.DecodeExpect[domain.EventMessage](bytes.NewReader([]byte{0x01})); err == nil {
		t.Fatal("DecodeExpect must propagate decode errors")
	}
}
