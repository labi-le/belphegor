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
