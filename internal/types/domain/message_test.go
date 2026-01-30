package domain_test

import (
	"testing"

	"github.com/labi-le/belphegor/internal/types/domain"
	"github.com/labi-le/belphegor/pkg/mime"
)

func TestMessage_Duplicate(t *testing.T) {
	tests := []struct {
		name string
		msg  domain.Message
		new  domain.Message
		want bool
	}{
		{
			name: "same message reference",
			msg:  domain.Message{ID: 1},
			new:  domain.Message{ID: 1},
			want: true,
		},
		{
			name: "same text content",
			msg:  domain.Message{ID: 1, ContentHash: 100, MimeType: mime.Type(1)},
			new:  domain.Message{ID: 2, ContentHash: 100, MimeType: mime.Type(1)},
			want: true,
		},
		{
			name: "different text content",
			msg:  domain.Message{ID: 1, ContentHash: 100, MimeType: mime.Type(1)},
			new:  domain.Message{ID: 2, ContentHash: 200, MimeType: mime.Type(1)},
			want: false,
		},
		{
			name: "same image different source",
			msg:  domain.Message{ID: 1, ContentHash: 500, MimeType: mime.Type(2)},
			new:  domain.Message{ID: 2, ContentHash: 500, MimeType: mime.Type(2)},
			want: true,
		},
		{
			name: "different images",
			msg:  domain.Message{ID: 1, ContentHash: 500, MimeType: mime.Type(2)},
			new:  domain.Message{ID: 2, ContentHash: 600, MimeType: mime.Type(2)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.msg.Duplicate(tt.new); got != tt.want {
				t.Errorf("%s: Content.Duplicate() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func BenchmarkMessage_Duplicate(b *testing.B) {
	benchmarks := []struct {
		name     string
		msg, new domain.Message
	}{
		{
			name: "same_id",
			msg:  domain.Message{ID: 1},
			new:  domain.Message{ID: 1},
		},
		{
			name: "different_id_same_hash",
			msg:  domain.Message{ID: 1, ContentHash: 123456789, MimeType: mime.Type(1)},
			new:  domain.Message{ID: 2, ContentHash: 123456789, MimeType: mime.Type(1)},
		},
		{
			name: "different_id_different_hash",
			msg:  domain.Message{ID: 1, ContentHash: 111111111, MimeType: mime.Type(1)},
			new:  domain.Message{ID: 2, ContentHash: 222222222, MimeType: mime.Type(1)},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				bm.msg.Duplicate(bm.new)
			}
		})
	}
}
