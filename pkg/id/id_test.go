package id_test

import (
	"testing"

	"github.com/labi-le/belphegor/pkg/id"
)

func TestAuthor(t *testing.T) {
	// snowflake layout: author (node) occupies bits 12..21 (10 bits, 0..1023).
	tests := []struct {
		name   string
		id     id.Unique
		author id.Unique
	}{
		{"zero", 0, 0},
		{"author 5 with low bits", (5 << 12) | 0xABC, 5},
		{"author 1023 max", (1023 << 12) | 0x1, 1023},
		{"only low bits set", 0xFFF, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := id.Author(tt.id); got != tt.author {
				t.Fatalf("Author(%d) = %d, want %d", tt.id, got, tt.author)
			}
		})
	}
}

func TestMine(t *testing.T) {
	self := (id.MyID << 12) | 0x42
	if !id.Mine(self) {
		t.Fatalf("Mine(id with author=MyID=%d) = false, want true", id.MyID)
	}

	other := (((id.MyID + 1) & 0x3FF) << 12) | 0x42
	if id.Mine(other) {
		t.Fatal("Mine(id from another author) = true, want false")
	}
}

func TestNew_UniqueAndAttributedToSelf(t *testing.T) {
	const n = 1000
	seen := make(map[id.Unique]struct{}, n)

	for range n {
		got := id.New()
		if _, dup := seen[got]; dup {
			t.Fatalf("New() produced duplicate id %d", got)
		}
		seen[got] = struct{}{}

		if !id.Mine(got) {
			t.Fatalf("New() id %d not attributed to MyID %d", got, id.MyID)
		}
	}
}
